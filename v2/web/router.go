package web

import (
	"crypto/rand"
	"database/sql"
	"net/http"

	"github.com/cloudfoundry-incubator/notifications/cf"
	"github.com/cloudfoundry-incubator/notifications/db"
	"github.com/cloudfoundry-incubator/notifications/gobble"
	"github.com/cloudfoundry-incubator/notifications/metrics"
	"github.com/cloudfoundry-incubator/notifications/uaa"
	"github.com/cloudfoundry-incubator/notifications/util"
	"github.com/cloudfoundry-incubator/notifications/v2/collections"
	"github.com/cloudfoundry-incubator/notifications/v2/models"
	"github.com/cloudfoundry-incubator/notifications/v2/queue"
	"github.com/cloudfoundry-incubator/notifications/v2/web/campaigns"
	"github.com/cloudfoundry-incubator/notifications/v2/web/campaigntypes"
	"github.com/cloudfoundry-incubator/notifications/v2/web/info"
	"github.com/cloudfoundry-incubator/notifications/v2/web/middleware"
	"github.com/cloudfoundry-incubator/notifications/v2/web/root"
	"github.com/cloudfoundry-incubator/notifications/v2/web/senders"
	"github.com/cloudfoundry-incubator/notifications/v2/web/templates"
	"github.com/cloudfoundry-incubator/notifications/v2/web/unsubscribers"
	"github.com/gorilla/mux"
	"github.com/pivotal-cf-experimental/rainmaker"
	"github.com/pivotal-cf-experimental/warrant"
	"github.com/pivotal-golang/lager"
	"github.com/ryanmoran/stack"
)

type muxer interface {
	Handle(method, path string, handler stack.Handler, middleware ...stack.Middleware)
	GetRouter() *mux.Router
	ServeHTTP(w http.ResponseWriter, req *http.Request)
}

type mother interface {
	SQLDatabase()
}

type enqueuer interface {
	Enqueue(job gobble.Job) (gobble.Job, error)
	EnqueueWithTransaction(job gobble.Job, transaction db.TransactionInterface) (gobble.Job, error)
}

type Config struct {
	DBLoggingEnabled bool
	SkipVerifySSL    bool
	SQLDB            *sql.DB
	Logger           lager.Logger
	Queue            enqueuer

	UAAPublicKey    string
	UAAHost         string
	UAAClientID     string
	UAAClientSecret string
	CCHost          string
}

func NewRouter(mx muxer, config Config) http.Handler {
	clock := util.NewClock()
	guidGenerator := util.NewIDGenerator(rand.Reader)

	requestCounter := middleware.NewRequestCounter(mx.GetRouter(), metrics.DefaultLogger)
	requestLogging := middleware.NewRequestLogging(config.Logger, clock)
	notificationsWriteAuthenticator := middleware.NewAuthenticator(config.UAAPublicKey, "notifications.write")
	notificationsAdminAuthenticator := middleware.NewAuthenticator(config.UAAPublicKey, "notifications.admin")
	unsubscribesAuthenticator := middleware.NewUnsubscribesAuthenticator(config.UAAPublicKey)
	databaseAllocator := middleware.NewDatabaseAllocator(config.SQLDB, config.DBLoggingEnabled)

	warrantConfig := warrant.Config{
		Host:          config.UAAHost,
		SkipVerifySSL: config.SkipVerifySSL,
	}
	warrantUsersService := warrant.NewUsersService(warrantConfig)
	warrantClientsService := warrant.NewClientsService(warrantConfig)

	rainmakerConfig := rainmaker.Config{
		Host:          config.CCHost,
		SkipVerifySSL: config.SkipVerifySSL,
	}
	rainmakerSpacesService := rainmaker.NewSpacesService(rainmakerConfig)
	rainmakerOrganizationsService := rainmaker.NewOrganizationsService(rainmakerConfig)

	userFinder := uaa.NewUserFinder(config.UAAClientID, config.UAAClientSecret, warrantUsersService, warrantClientsService)
	spaceFinder := cf.NewSpaceFinder(config.UAAClientID, config.UAAClientSecret, warrantClientsService, rainmakerSpacesService)
	orgFinder := cf.NewOrgFinder(config.UAAClientID, config.UAAClientSecret, warrantClientsService, rainmakerOrganizationsService)

	campaignEnqueuer := queue.NewCampaignEnqueuer(config.Queue)

	sendersRepository := models.NewSendersRepository(guidGenerator.Generate)
	campaignTypesRepository := models.NewCampaignTypesRepository(guidGenerator.Generate)
	templatesRepository := models.NewTemplatesRepository(guidGenerator.Generate)
	campaignsRepository := models.NewCampaignsRepository(guidGenerator.Generate)
	messagesRepository := models.NewMessagesRepository(clock, guidGenerator.Generate)
	unsubscribersRepository := models.NewUnsubscribersRepository(guidGenerator.Generate)

	sendersCollection := collections.NewSendersCollection(sendersRepository, campaignTypesRepository)
	templatesCollection := collections.NewTemplatesCollection(templatesRepository)
	campaignTypesCollection := collections.NewCampaignTypesCollection(campaignTypesRepository, sendersRepository, templatesRepository)
	campaignsCollection := collections.NewCampaignsCollection(campaignEnqueuer, campaignsRepository, campaignTypesRepository, templatesRepository, sendersRepository, userFinder, spaceFinder, orgFinder)
	campaignStatusesCollection := collections.NewCampaignStatusesCollection(campaignsRepository, sendersRepository, messagesRepository)
	unsubscribersCollection := collections.NewUnsubscribersCollection(unsubscribersRepository, campaignTypesRepository, userFinder)

	root.Routes{
		RequestLogging: requestLogging,
	}.Register(mx)

	info.Routes{
		RequestCounter: requestCounter,
		RequestLogging: requestLogging,
	}.Register(mx)

	senders.Routes{
		RequestLogging:    requestLogging,
		Authenticator:     notificationsWriteAuthenticator,
		DatabaseAllocator: databaseAllocator,
		SendersCollection: sendersCollection,
	}.Register(mx)

	campaigntypes.Routes{
		RequestLogging:          requestLogging,
		Authenticator:           notificationsWriteAuthenticator,
		DatabaseAllocator:       databaseAllocator,
		CampaignTypesCollection: campaignTypesCollection,
	}.Register(mx)

	templates.Routes{
		RequestLogging:      requestLogging,
		WriteAuthenticator:  notificationsWriteAuthenticator,
		AdminAuthenticator:  notificationsAdminAuthenticator,
		DatabaseAllocator:   databaseAllocator,
		TemplatesCollection: templatesCollection,
	}.Register(mx)

	campaigns.Routes{
		Clock:                      clock,
		RequestLogging:             requestLogging,
		Authenticator:              notificationsWriteAuthenticator,
		DatabaseAllocator:          databaseAllocator,
		CampaignsCollection:        campaignsCollection,
		CampaignStatusesCollection: campaignStatusesCollection,
	}.Register(mx)

	unsubscribers.Routes{
		RequestLogging:          requestLogging,
		Authenticator:           unsubscribesAuthenticator,
		DatabaseAllocator:       databaseAllocator,
		UnsubscribersCollection: unsubscribersCollection,
	}.Register(mx)

	return mx
}
