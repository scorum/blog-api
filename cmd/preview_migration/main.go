package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"time"

	"github.com/Azure/azure-storage-blob-go/2016-05-31/azblob"
	"gitlab.scorum.com/blog/api/common"
	"gitlab.scorum.com/blog/api/service"
	"gitlab.scorum.com/blog/api/service/image"
)

var (
	azureStorageAccount = flag.String("azure_storage_account", "", "azure storage account, required")
	azureAccessKey      = flag.String("azure_access_key", "", "azure access key, required")
	azureContainerName  = flag.String("azure_container_name", "", "azure container name, required")

	fromFlag = flag.String("from",
		time.Date(0, time.January, 1, 0, 0, 0, 0, time.UTC).Format(service.TimeLayout),
		"min modified_at value for blobs")
)

// utility adds a preview thumbnail for the existing media resources
// usage: ./preview_migration -azure_storage_account="" -azure_access_key="" -azure_container_name=""
func main() {
	flag.Parse()

	if *azureStorageAccount == "" || *azureAccessKey == "" || *azureContainerName == "" {
		flag.Usage()
		os.Exit(1)
	}

	from, err := time.Parse(service.TimeLayout, *fromFlag)
	if err != nil {
		flag.Usage()
		os.Exit(1)
	}

	// Create a default request pipeline using your storage account name and account key.
	credential := azblob.NewSharedKeyCredential(*azureStorageAccount, *azureAccessKey)
	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})

	// From the Azure portal, get your storage account blob service URL endpoint.
	URL, _ := url.Parse(
		fmt.Sprintf("https://%s.blob.core.windows.net/%s", *azureStorageAccount, *azureContainerName))

	// Create a ContainerURL object that wraps the container URL and a request
	// pipeline to make requests.
	containerURL := azblob.NewContainerURL(*URL, p)

	ctx := context.Background()

	// List the blobs in the container
	for marker := (azblob.Marker{}); marker.NotDone(); {
		// Get a result segment starting with the blob indicated by the current Marker.
		listBlob, err := containerURL.ListBlobs(ctx, marker, azblob.ListBlobsOptions{})
		if err != nil {
			log.Fatalf("failed to list blob: %s", err)
		}

		// ListBlobs returns the start of the next segment; you MUST use this to get
		// the next segment (after processing the current result segment).
		marker = listBlob.NextMarker

		// Process the blobs returned in this result segment (if the segment is empty, the loop body won't execute)
		for _, blobInfo := range listBlob.Blobs.Blob {
			name := blobInfo.Name

			// Original image does not contain _, dirty but works
			if strings.Contains(name, "_") {
				continue
			}

			// skip already migrated blobs
			if blobInfo.Properties.LastModified.Before(from) {
				continue
			}

			log.Print("blob name: " + name + "\n")
			original := containerURL.NewBlockBlobURL(name)

			// Download blob
			stream := azblob.NewDownloadStream(ctx, original.GetBlob, azblob.DownloadStreamOptions{})
			downloadedData := &bytes.Buffer{}
			_, err = downloadedData.ReadFrom(stream)
			if err != nil {
				log.Printf("failed to download %s: %s\n", name, err)
				continue
			}

			img, err := image.NewImage(downloadedData.Bytes(), common.ContentType(*blobInfo.Properties.ContentType))
			if err != nil {
				log.Printf("failed to create image %s: %s\n", name, err)
				continue
			}

			// Preview
			service.AddPreviewToImage(img)

			// Wide preview
			service.AddPreviewWideToImage(img)

			// High preview
			service.AddPreviewHighToImage(img)

			// Notification
			service.AddNotificationPreviewToImage(img)

			// Profile
			service.AddProfilePreviewToImage(img)

			for _, preview := range img.Thumbs {
				// Encode preview
				var content bytes.Buffer

				if err := img.Encode(&content, preview); err != nil {
					log.Printf("failed to encode image %s_%s: %s\n", name, preview.Postfix, err)
					continue
				}

				// Upload preview
				blobUrl := containerURL.NewBlockBlobURL(fmt.Sprintf("%s_%s", name, preview.Postfix))

				_, err = blobUrl.PutBlob(
					context.Background(),
					bytes.NewReader(content.Bytes()), azblob.BlobHTTPHeaders{
						ContentType: string(img.ContentType),
					}, azblob.Metadata{}, azblob.BlobAccessConditions{})

				if err != nil {
					log.Printf("failed to upload image %s_%s: %s\n", name, preview.Postfix, err)
				}
			}
		}
	}
}
