package aws_api

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
)

type Route53API struct {
	svc         *route53.Route53
	profileName *string
}

func Route53APINew(profileName *string) *Route53API {
	if profileName == nil {
		profileNameString := "default"
		profileName = &profileNameString
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            aws.Config{Region: StrPtr("us-east-1")},
		Profile:           *profileName,
	}))

	lg.InfoF("AWS profile: %s\n", *profileName)
	svc := route53.New(sess)
	ret := Route53API{svc: svc, profileName: profileName}
	return &ret
}

// Up to 100 per page
func (api *Route53API) YieldHostedZones(Input *route53.ListHostedZonesInput, callbackFilter GenericCallback) ([]*route53.HostedZone, error) {
	var callbackErr error
	ret := []*route53.HostedZone{}

	err := api.svc.ListHostedZonesPages(Input, func(page *route53.ListHostedZonesOutput, notHasNextPage bool) bool {
		for _, hz := range page.HostedZones {
			if callbackFilter != nil {
				if callbackErr = callbackFilter(hz); callbackErr != nil {
					return false
				}
			}
			ret = append(ret, hz)
		}
		return true
	})

	if callbackErr != nil {
		return nil, callbackErr
	}
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (api *Route53API) YieldResourceRecordSets(Input *route53.ListResourceRecordSetsInput) ([]*route53.ResourceRecordSet, error) {
	inputs := []*route53.ListResourceRecordSetsInput{}
	if Input == nil {
		hzs, err := api.YieldHostedZones(nil, nil)
		if err != nil {
			return nil, err
		}
		for _, hz := range hzs {
			inputs = append(inputs, &route53.ListResourceRecordSetsInput{HostedZoneId: hz.Id})
		}

	} else {
		inputs = append(inputs, Input)
	}

	ret := []*route53.ResourceRecordSet{}
	for _, input := range inputs {
		rcrds, err := api.YieldHostedZoneResourceRecordSets(input)
		if err != nil {
			return nil, err
		}
		ret = append(ret, rcrds...)
	}

	return ret, nil
}

func (api *Route53API) YieldHostedZoneResourceRecordSets(Input *route53.ListResourceRecordSetsInput) ([]*route53.ResourceRecordSet, error) {
	ret := []*route53.ResourceRecordSet{}

	err := api.svc.ListResourceRecordSetsPages(Input, func(page *route53.ListResourceRecordSetsOutput, notHasNextPage bool) bool {
		ret = append(ret, page.ResourceRecordSets...)
		return true
	})

	if err != nil {
		return nil, err
	}

	return ret, nil
}
