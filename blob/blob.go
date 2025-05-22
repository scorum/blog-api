package blob

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/Azure/azure-storage-blob-go/2016-05-31/azblob"
	"github.com/sirupsen/logrus"
	"gitlab.scorum.com/blog/api/common"
)

const blobFormatString = `https://%s.blob.core.windows.net`

type Service struct {
	config Config
}

func NewService(cfg Config) *Service {
	service := &Service{
		config: cfg,
	}

	// create container if not exists
	container := service.getContainerURL(cfg.Container)
	_, err := container.Create(context.Background(), azblob.Metadata{}, azblob.PublicAccessContainer)
	if err != nil {
		if serr, ok := err.(azblob.StorageError); ok {
			if serr.ServiceCode() == azblob.ServiceCodeContainerAlreadyExists {
				return service
			}
		}
		logrus.Fatal(err)
	}
	return service
}

func (s *Service) primaryUrl() string {
	return fmt.Sprintf(blobFormatString, s.config.AccountName)
}

// serviceURL returns url representing a URL to the Azure Storage Blob service
// allowing you to manipulate blob containers.
func (s *Service) serviceURL() azblob.ServiceURL {
	c := azblob.NewSharedKeyCredential(s.config.AccountName, s.config.AccountKey)
	pipeline := azblob.NewPipeline(c, azblob.PipelineOptions{})
	u, _ := url.Parse(s.primaryUrl())
	return azblob.NewServiceURL(*u, pipeline)
}

func (s *Service) getContainerURL(containerName string) azblob.ContainerURL {
	return s.serviceURL().NewContainerURL(containerName)
}

// UploadMedia uploads media content to the Azure blob
func (s *Service) UploadMedia(account string, ID string, content []byte, contentType common.ContentType) (string, error) {
	container := s.getContainerURL(s.config.Container)
	blobUrl := container.NewBlockBlobURL(fmt.Sprintf("%s/%s", account, ID))

	_, err := blobUrl.PutBlob(
		context.Background(),
		bytes.NewReader(content), azblob.BlobHTTPHeaders{
			ContentType: string(contentType),
		}, azblob.Metadata{}, azblob.BlobAccessConditions{})
	if err != nil {
		return "", err
	}

	url := blobUrl.URL()
	return strings.Replace((&url).String(), s.primaryUrl(), s.config.CDNDomain, 1), nil
}

func (s *Service) DoesMediaExists(account, ID string) (bool, error) {
	container := s.getContainerURL(s.config.Container)
	blobUrl := container.NewBlockBlobURL(fmt.Sprintf("%s/%s", account, ID))

	_, err := blobUrl.GetPropertiesAndMetadata(context.Background(), azblob.BlobAccessConditions{})

	if err != nil {
		if serr, ok := err.(azblob.StorageError); ok {
			// there is no other way to identify BlobNotFound error
			if strings.Contains(serr.Error(), "404 The specified blob does not exist.") {
				return false, nil
			}
		}
		return false, err
	}

	return true, nil
}
