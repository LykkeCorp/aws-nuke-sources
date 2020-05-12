package resources

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/LykkeCorp/aws-nuke-sources/pkg/types"
)

type ELBv2LoadBalancer struct {
	svc  *elbv2.ELBV2
	name *string
	arn  *string
	tags []*elbv2.Tag
}

func init() {
	register("ELBv2", ListELBv2LoadBalancers)
}

func ListELBv2LoadBalancers(sess *session.Session) ([]Resource, error) {
	svc := elbv2.New(sess)

	elbResp, err := svc.DescribeLoadBalancers(nil)
	if err != nil {
		return nil, err
	}

	var tagReqELBv2ARNs []*string
	ELBv2ArnToName := make(map[string]*string)
	for _, elbv2 := range elbResp.LoadBalancers {
		tagReqELBv2ARNs = append(tagReqELBv2ARNs, elbv2.LoadBalancerArn)
		ELBv2ArnToName[*elbv2.LoadBalancerArn] = elbv2.LoadBalancerName
	}

	// Tags for ELBv2s need to be fetched separately
	// We can only specify up to 20 in a single call
	// See: https://github.com/aws/aws-sdk-go/blob/0e8c61841163762f870f6976775800ded4a789b0/service/elbv2/api.go#L5398
	resources := make([]Resource, 0)
	for len(tagReqELBv2ARNs) > 0 {
		requestElements := len(tagReqELBv2ARNs)
		if requestElements > 20 {
			requestElements = 20
		}

		tagResp, err := svc.DescribeTags(&elbv2.DescribeTagsInput{
			ResourceArns: tagReqELBv2ARNs[:requestElements],
		})
		if err != nil {
			return nil, err
		}
		for _, elbv2TagInfo := range tagResp.TagDescriptions {
			resources = append(resources, &ELBv2LoadBalancer{
				svc:  svc,
				name: ELBv2ArnToName[*elbv2TagInfo.ResourceArn],
				arn:  elbv2TagInfo.ResourceArn,
				tags: elbv2TagInfo.Tags,
			})
		}

		// Remove the elements that were queried
		tagReqELBv2ARNs = tagReqELBv2ARNs[requestElements:]
	}
	return resources, nil
}

func (e *ELBv2LoadBalancer) Remove() error {
	params := &elbv2.DeleteLoadBalancerInput{
		LoadBalancerArn: e.arn,
	}

	_, err := e.svc.DeleteLoadBalancer(params)
	if err != nil {
		return err
	}

	return nil
}

func (e *ELBv2LoadBalancer) Properties() types.Properties {
	properties := types.NewProperties()
	for _, tagValue := range e.tags {
		properties.SetTag(tagValue.Key, tagValue.Value)
	}
	return properties
}

func (e *ELBv2LoadBalancer) String() string {
	return *e.name
}
