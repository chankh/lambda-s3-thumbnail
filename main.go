package main

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/disintegration/imaging"
	log "github.com/sirupsen/logrus"
)

// temp location to store image and thumbnail
const tmp = "/tmp/"

// S3 Session to use
var sess = session.Must(session.NewSession())

// Create an uploader with session and default option
var uploader = s3manager.NewUploader(sess)

// Create a downloader with session and default option
var downloader = s3manager.NewDownloader(sess)

func handle(ctx context.Context, req events.S3Event) (string, error) {
	log.SetOutput(os.Stdout)
	log.Infof("%v", req)
	for _, r := range req.Records {
		if key := r.S3.Object.Key; isImage(key) {
			// generate thumbnail
			bucket := r.S3.Bucket.Name
			genThumb(bucket, key)
		}
	}
	return fmt.Sprintf("%d records processed", len(req.Records)), nil
}

func genThumb(bucket, key string) {
	local := tmp + bucket + "/" + key

	// ensure path is available
	dir := filepath.Dir(local)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		log.WithError(err).WithField("path", dir).Error("failed to create tmp directory")
	}

	// create a file locally for original image in S3
	f, err := os.Create(local)
	if err != nil {
		log.WithError(err).WithField("filename", local).Error("failed to create file")
		return
	}

	n, err := downloader.Download(f, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"bucket":   bucket,
			"key":      key,
			"filename": local,
		}).Error("failed to download file")
		return
	}

	log.WithFields(log.Fields{
		"filename": local,
		"bytes":    n,
	}).Info("file downloaded")

	img, err := imaging.Open(local)
	if err != nil {
		panic(err)
	}
	thumb := imaging.Thumbnail(img, 100, 100, imaging.CatmullRom)

	// create a new blank image
	dst := imaging.New(100, 100, color.NRGBA{0, 0, 0, 0})

	// paste thumbnails into the new image
	dst = imaging.Paste(dst, thumb, image.Pt(0, 0))

	// save the combined image to file
	thumbName := key[:len(key)-4] + "_thumb" + key[len(key)-4:]
	thumbLocal := "/tmp/" + thumbName
	err = imaging.Save(dst, thumbLocal)
	if err != nil {
		log.WithError(err).WithField("thumbnail", thumbLocal).Error("failed to generate thumbnail")
		return
	}

	// upload thumbnail to S3
	up, err := os.Open(thumbLocal)
	if err != nil {
		log.WithError(err).WithField("thumbnail", thumbLocal).Error("failed to open file")
		return
	}

	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(thumbName),
		Body:   up,
	})

	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"bucket":    bucket,
			"thumbnail": thumbName,
		}).Error("failed to upload file")
	}

	log.WithField("location", result.Location).Info("file uploaded")
}

func isImage(name string) bool {
	if strings.HasSuffix(name, ".jpg") {
		return true
	}

	if strings.HasSuffix(name, ".png") {
		return true
	}

	if strings.HasSuffix(name, ".gif") {
		return true
	}

	return false
}

func main() {
	lambda.Start(handle)
}
