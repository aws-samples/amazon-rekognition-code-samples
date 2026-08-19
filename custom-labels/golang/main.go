package main

import (
	"context"
	"flag"
	"image"

	_ "image/jpeg"
	_ "image/png"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/fogleman/gg"
)

var bucketName = flag.String("bucket", "", "The name of the bucket to get the object")
var key = flag.String("key", "", "The s3 object key to the image file (JPEG, JPG, PNG)")
var modelArn = flag.String("model-arn", "", "The rekognition custom labels model arn")
var minConfidence = float32(*flag.Float64("min-confidence", 70, "The minimum confidence value "))

func main() {
    flag.Parse()

    cfg, err := config.LoadDefaultConfig(context.TODO())
    if err != nil {
        log.Fatalf("unable to load SDK config, %v", err)
    }

    svc := rekognition.NewFromConfig(cfg)
    var s3Obj = types.S3Object{
        Bucket: bucketName,
        Name:   key}

    resp, err := svc.DetectCustomLabels(context.TODO(), &rekognition.DetectCustomLabelsInput{
        Image: &types.Image{
            S3Object: &s3Obj},
        ProjectVersionArn: modelArn,
        MinConfidence:     aws.Float32(minConfidence),
    })

    if err != nil {
        log.Fatalf("unable to detect custom labels, %v", err)
    }

    log.Printf("Detected %v labels", len(resp.CustomLabels))

    for _, label := range resp.CustomLabels {
        log.Printf("%v: %v\b", *label.Name, *label.Confidence)
    }

    s3Client := s3.NewFromConfig(cfg)
    input := s3.GetObjectInput{
        Bucket: bucketName,
        Key:    key,
    }
    s3Resp, err := s3Client.GetObject(context.TODO(), &input)
    if err != nil {
        log.Fatalf("unable to retrieve image: %v", err)
    }

    i, _, err := image.Decode(s3Resp.Body)
    if err != nil {
        log.Fatalf("can't decode image: %v", err)
    }

    ggContext := gg.NewContextForImage(i)
    err = ggContext.LoadFontFace("/Library/Fonts/Arial Unicode.ttf", 100)
    if err != nil {
        log.Fatalf("Unable to load font: %v", err)
    }
    ggContext.SetRGB(1, 0, 0)
    ggContext.SetLineWidth(10)

    for i, label := range resp.CustomLabels {
        x := float64(*label.Geometry.BoundingBox.Left) * float64(ggContext.Width())
        y := float64(*label.Geometry.BoundingBox.Top) * float64(ggContext.Height())
        w := float64(*label.Geometry.BoundingBox.Width) * float64(ggContext.Width())
        h := float64(*label.Geometry.BoundingBox.Height) * float64(ggContext.Height())

        log.Printf("Detected label #%v: %v", i, *label.Name)
        log.Printf("\tLeft: %v", x)
        log.Printf("\tTop: %v", y)
        log.Printf("\tWidth: %v", w)
        log.Printf("\tHeight: %v", h)

        ggContext.DrawRectangle(x, y, w, h)
        ggContext.Stroke()
        ggContext.DrawString(*label.Name, x, y)
        ggContext.Stroke()
    }

    err = ggContext.SavePNG("output.png")
    if err != nil {
        log.Fatalf("error saving image: %v", err)
    }
}
