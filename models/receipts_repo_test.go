package models_test

import (
    "time"

    "github.com/cloudfoundry-incubator/notifications/models"
    "github.com/coopernurse/gorp"

    . "github.com/onsi/ginkgo"
    . "github.com/onsi/gomega"
)

var _ = Describe("Receipts Repo", func() {
    var repo models.ReceiptsRepo
    var conn *gorp.DbMap

    BeforeEach(func() {
        TruncateTables()
        repo = models.NewReceiptsRepo()
        conn = models.Database().Connection
    })

    Describe("Create", func() {
        var receipt models.Receipt

        BeforeEach(func() {
            receipt = models.Receipt{
                UserGUID: "user-123",
                ClientID: "client-abc",
                KindID:   "abc-def",
            }
        })

        It("stores the receipt in the database", func() {
            receipt, err := repo.Create(conn, receipt)
            if err != nil {
                panic(err)
            }

            receipt, err = repo.Find(conn, receipt.UserGUID, receipt.ClientID, receipt.KindID)
            if err != nil {
                panic(err)
            }

            Expect(receipt.UserGUID).To(Equal("user-123"))
            Expect(receipt.ClientID).To(Equal("client-abc"))
            Expect(receipt.KindID).To(Equal("abc-def"))
            Expect(receipt.Count).To(Equal(1))
            Expect(receipt.CreatedAt).To(BeTemporally("~", time.Now(), 2*time.Second))
        })

        It("returns an ErrDuplicateRecord when the receipt already exists in the database", func() {
            _, err := repo.Create(conn, receipt)
            if err != nil {
                panic(err)
            }

            receipt, err = repo.Create(conn, receipt)
            Expect(err).To(Equal(models.ErrDuplicateRecord{}))
        })
    })

    Describe("Find", func() {
        It("returns a receipt when it exists in the database", func() {
            receipt := models.Receipt{
                UserGUID: "user-123",
                ClientID: "client-abc",
                KindID:   "abc-def",
            }

            receipt, err := repo.Create(conn, receipt)
            if err != nil {
                panic(err)
            }

            receipt, err = repo.Find(conn, receipt.UserGUID, receipt.ClientID, receipt.KindID)
            if err != nil {
                panic(err)
            }

            Expect(receipt.UserGUID).To(Equal("user-123"))
            Expect(receipt.ClientID).To(Equal("client-abc"))
            Expect(receipt.KindID).To(Equal("abc-def"))
            Expect(receipt.Count).To(Equal(1))
            Expect(receipt.CreatedAt).To(BeTemporally("~", time.Now(), 2*time.Second))
        })

        It("returns an ErrRecordNotFound when the requested receipt does not exist in the database", func() {
            _, err := repo.Find(conn, "user-000", "client-000", "unkind-client")
            Expect(err).To(Equal(models.ErrRecordNotFound{}))
        })
    })

    Describe("Update", func() {
        It("updates the receipt in the database", func() {
            receipt := models.Receipt{
                UserGUID: "user-123",
                ClientID: "client-abc",
                KindID:   "abc-def",
            }

            receipt, err := repo.Create(conn, receipt)
            if err != nil {
                panic(err)
            }

            receipt.KindID = "be-kind"
            repo.Update(conn, receipt)

            receipt, err = repo.Find(conn, receipt.UserGUID, receipt.ClientID, receipt.KindID)
            if err != nil {
                panic(err)
            }

            Expect(receipt.UserGUID).To(Equal("user-123"))
            Expect(receipt.ClientID).To(Equal("client-abc"))
            Expect(receipt.KindID).To(Equal("be-kind"))
            Expect(receipt.Count).To(Equal(1))
            Expect(receipt.CreatedAt).To(BeTemporally("~", time.Now(), 2*time.Second))
        })
    })

    Describe("Upsert", func() {
        var receipt models.Receipt
        userGUID := "user-123"
        clientID := "client-abc"
        kindID := "abc-def"

        BeforeEach(func() {
            receipt = models.Receipt{
                UserGUID: userGUID,
                ClientID: clientID,
                KindID:   kindID,
            }
        })

        It("creates a receipt in the database when no matching record exists", func() {
            receipt, err := repo.Upsert(conn, receipt)
            if err != nil {
                panic(err)
            }

            receipt, err = repo.Find(conn, receipt.UserGUID, receipt.ClientID, receipt.KindID)
            if err != nil {
                panic(err)
            }

            Expect(receipt.UserGUID).To(Equal(userGUID))
            Expect(receipt.ClientID).To(Equal(clientID))
            Expect(receipt.KindID).To(Equal(kindID))
            Expect(receipt.Count).To(Equal(1))
            Expect(receipt.CreatedAt).To(BeTemporally("~", time.Now(), 2*time.Second))
        })

        It("auto-increments the count when a record with same user, client, kind is inserted", func() {
            receipt, err := repo.Create(conn, receipt)
            if err != nil {
                panic(err)
            }

            second_receipt, err := repo.Upsert(conn, receipt)
            if err != nil {
                panic(err)
            }

            Expect(receipt.Count).To(Equal(1))
            Expect(second_receipt.Count).To(Equal(2))
            Expect(receipt.Primary).To(Equal(second_receipt.Primary))
        })
    })

    Describe("CreateReceipts", func() {
        var firstUserGUID string
        var secondUserGUID string
        var userGUIDs []string
        var clientID string
        var kindID string

        BeforeEach(func() {
            firstUserGUID = "user-123"
            secondUserGUID = "user-456"
            userGUIDs = []string{firstUserGUID, secondUserGUID}
            clientID = "client-abc"
            kindID = "be-kind"
        })

        It("creates or updates a receipt for each user", func() {
            err := repo.CreateReceipts(conn, userGUIDs, clientID, kindID)
            if err != nil {
                panic(err)
            }

            firstReceipt, err := repo.Find(conn, firstUserGUID, clientID, kindID)
            if err != nil {
                panic(err)
            }

            secondReceipt, err := repo.Find(conn, secondUserGUID, clientID, kindID)
            if err != nil {
                panic(err)
            }

            Expect(firstReceipt.UserGUID).To(Equal(firstUserGUID))
            Expect(firstReceipt.ClientID).To(Equal(clientID))
            Expect(firstReceipt.KindID).To(Equal(kindID))
            Expect(firstReceipt.Count).To(Equal(1))

            Expect(secondReceipt.UserGUID).To(Equal(secondUserGUID))
            Expect(secondReceipt.ClientID).To(Equal(clientID))
            Expect(secondReceipt.KindID).To(Equal(kindID))
            Expect(secondReceipt.Count).To(Equal(1))
        })

        It("updates a receipt's count for a user for a given clientID and kindID", func() {
            receipt := models.Receipt{
                UserGUID: firstUserGUID,
                ClientID: clientID,
                KindID:   kindID,
            }

            _, err := repo.Create(conn, receipt)
            if err != nil {
                panic(err)
            }

            rowCount, err := conn.SelectInt("SELECT COUNT(*) FROM `receipts`")
            if err != nil {
                panic(err)
            }

            Expect(int(rowCount)).To(Equal(1))

            err = repo.CreateReceipts(conn, []string{firstUserGUID}, clientID, kindID)
            if err != nil {
                panic(err)
            }

            rowCount, err = conn.SelectInt("SELECT COUNT(*) FROM `receipts`")
            if err != nil {
                panic(err)
            }

            Expect(int(rowCount)).To(Equal(1))

            firstReceipt, err := repo.Find(conn, firstUserGUID, clientID, kindID)
            if err != nil {
                panic(err)
            }

            Expect(firstReceipt.UserGUID).To(Equal(firstUserGUID))
            Expect(firstReceipt.ClientID).To(Equal(clientID))
            Expect(firstReceipt.KindID).To(Equal(kindID))
            Expect(firstReceipt.Count).To(Equal(2))
        })

        It("does not update count and adds a row when clientID and kindID are different", func() {
            _, err := repo.Create(conn, models.Receipt{
                UserGUID: firstUserGUID,
                ClientID: clientID,
                KindID:   kindID,
            })

            rowCount, err := conn.SelectInt("SELECT COUNT(*) FROM `receipts`")
            if err != nil {
                panic(err)
            }

            Expect(int(rowCount)).To(Equal(1))

            err = repo.CreateReceipts(conn, []string{firstUserGUID}, "weird-client", kindID)
            if err != nil {
                panic(err)
            }

            rowCount, err = conn.SelectInt("SELECT COUNT(*) FROM `receipts`")
            if err != nil {
                panic(err)
            }
            Expect(int(rowCount)).To(Equal(2))

            err = repo.CreateReceipts(conn, []string{firstUserGUID}, clientID, "a-new-kind")
            if err != nil {
                panic(err)
            }

            rowCount, err = conn.SelectInt("SELECT COUNT(*) FROM `receipts`")
            if err != nil {
                panic(err)
            }
            Expect(int(rowCount)).To(Equal(3))

            firstReceipt, err := repo.Find(conn, firstUserGUID, clientID, kindID)
            if err != nil {
                panic(err)
            }

            differentClientReceipt, err := repo.Find(conn, firstUserGUID, "weird-client", kindID)
            if err != nil {
                panic(err)
            }

            differentKindReceipt, err := repo.Find(conn, firstUserGUID, clientID, "a-new-kind")
            if err != nil {
                panic(err)
            }

            Expect(firstReceipt.UserGUID).To(Equal(firstUserGUID))
            Expect(differentKindReceipt.UserGUID).To(Equal(firstUserGUID))
            Expect(differentClientReceipt.UserGUID).To(Equal(firstUserGUID))
            Expect(firstReceipt.Primary).ToNot(Equal(differentClientReceipt.Primary))
            Expect(firstReceipt.Primary).ToNot(Equal(differentKindReceipt.Primary))
        })
    })
})