package services

import "github.com/cloudfoundry-incubator/notifications/models"

type TemplateFinder struct {
	templatesRepo TemplatesRepo
}

type TemplateFinderInterface interface {
	FindByID(models.DatabaseInterface, string) (models.Template, error)
}

func NewTemplateFinder(templatesRepo TemplatesRepo) TemplateFinder {
	return TemplateFinder{
		templatesRepo: templatesRepo,
	}
}

func (finder TemplateFinder) FindByID(database models.DatabaseInterface, templateID string) (models.Template, error) {
	template, err := finder.templatesRepo.FindByID(database.Connection(), templateID)
	if err != nil {
		return models.Template{}, err
	}

	return template, err
}
