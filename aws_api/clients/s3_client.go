package aws_api

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type S3API struct {
	svc         *s3.S3
	region      *string
	profileName *string
}

func S3APINew(region *string, profileName *string) *S3API {
	if profileName == nil {
		profileNameString := "default"
		profileName = &profileNameString
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            aws.Config{Region: region},
		Profile:           *profileName,
	}))

	lg.Infof("AWS profile: %s\n", *profileName)
	svc := s3.New(sess)
	ret := S3API{svc: svc, region: region, profileName: profileName}
	return &ret
}

func (api *S3API) ListBuckets(callback GenericCallback, Input *s3.ListBucketsInput) error {
	var callbackErr error

	output, err := api.svc.ListBuckets(Input)
	if err != nil {
		return err
	}

	for _, obj := range output.Buckets {
		if callbackErr = callback(obj); callbackErr != nil {
			return callbackErr
		}
	}

	return nil
}

func (api *S3API) AddTags(AddTags map[string]string, bucket *s3.Bucket, declarative bool) (*s3.PutBucketTaggingOutput, error) {
	bucket_region, err := api.GetBucketRegion(bucket)
	if err != nil {
		return nil, err
	}

	if *api.region != *bucket_region {
		api = S3APINew(bucket_region, api.profileName)
	}

	existingTags, err := api.GetTags(bucket.Name)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchTagSet") {
			existingTags = []*s3.Tag{}
		} else {
			return nil, err
		}
	}

	Tagging := s3.Tagging{TagSet: []*s3.Tag{}}

	Tagging.TagSet = append(Tagging.TagSet, existingTags...)

	for key, value := range AddTags {
		found := false
		for _, existingTag := range existingTags {
			if *existingTag.Key == key && (*existingTag.Value == value || !declarative) {
				found = true
				break
			}
		}

		if !found {
			Tagging.TagSet = append(Tagging.TagSet, &s3.Tag{Key: &key, Value: &value})
		}
	}
	if len(Tagging.TagSet) == len(existingTags) {
		return nil, nil
	}
	lg.Infof("Adding tags: resource: %s, tags: %v, Current tags: %v", *bucket.Name, Tagging.TagSet, existingTags)

	createTagsOutput, err := api.svc.PutBucketTagging(&s3.PutBucketTaggingInput{Bucket: bucket.Name, Tagging: &Tagging})
	return createTagsOutput, err
}

func (api *S3API) GetBucketRegion(bucket *s3.Bucket) (region *string, err error) {
	varRegion := "us-east-1"
	region = &varRegion
	response, err := api.svc.GetBucketLocation(&s3.GetBucketLocationInput{Bucket: bucket.Name})
	if err != nil {
		return region, err
	}

	if response.LocationConstraint != nil {
		region = response.LocationConstraint
	}

	return region, err
}

func (api *S3API) GetTags(name *string) ([]*s3.Tag, error) {
	describeTagsOutput, err := api.svc.GetBucketTagging(&s3.GetBucketTaggingInput{Bucket: name})

	if err != nil {
		return nil, err
	}

	return describeTagsOutput.TagSet, nil

}
