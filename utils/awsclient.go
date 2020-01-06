package utils

import (
	"fmt"
  "strings"
  "regexp"
  "net"

	"github.com/urfave/cli/v2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
)

// AWSClientImpl ...
type AWSClientImpl struct {
  r53 Route53Client
}

// Route53Client ...
type Route53Client interface {
  ListHostedZonesByName(input *route53.ListHostedZonesByNameInput) (*route53.ListHostedZonesByNameOutput, error)
  ListResourceRecordSets(input *route53.ListResourceRecordSetsInput) (*route53.ListResourceRecordSetsOutput, error)
  ChangeResourceRecordSets(input *route53.ChangeResourceRecordSetsInput) (*route53.ChangeResourceRecordSetsOutput, error)
}

// ReverseHostedZoneInfos ...
type ReverseHostedZoneInfos []ReverseHostedZoneInfo

// ReverseHostedZoneInfo ...
type ReverseHostedZoneInfo struct {
  Network *net.IPNet
  NetworkCIDR string
  HostedZoneID string
  HostedZoneName string
}

// NewAWSClient ...
func NewAWSClient(c *cli.Context) (*AWSClientImpl, error) {
  profileName := c.String("profile")
  config := aws.NewConfig()
  sessOpts := session.Options{
    Config: *config,
    Profile: profileName,
    AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
    SharedConfigState: session.SharedConfigEnable,
  }
  sess := session.Must(session.NewSessionWithOptions(sessOpts))
  return &AWSClientImpl{
    r53: route53.New(sess),
  }, nil
}

// GetHostedZoneID ...
func (client *AWSClientImpl) GetHostedZoneID(hostedZoneName string) (hostedZoneID string, err error) {
  input := route53.ListHostedZonesByNameInput{
    DNSName: aws.String(hostedZoneName),
    MaxItems: aws.String("1"),
  }

  var resp *route53.ListHostedZonesByNameOutput
  resp, err = client.r53.ListHostedZonesByName(&input)

  if err != nil {
    return "", fmt.Errorf("HostedZone not found: %s", hostedZoneName)
  }

  hostedZone := *resp.HostedZones[0]
  rHostedZoneName := aws.StringValue(hostedZone.Name)
  if compareHostedZoneName(hostedZoneName, rHostedZoneName) != true {
    return "", fmt.Errorf("unexpected HostedZone Name: expected %s, actual %s", hostedZoneName, rHostedZoneName)
  }

  hostedZoneIDParts := strings.Split(aws.StringValue(hostedZone.Id), "/")
  hostedZoneID = hostedZoneIDParts[len(hostedZoneIDParts)-1]

  return hostedZoneID, nil
}

// ListAllResourceRecords ...
func (client *AWSClientImpl) ListAllResourceRecords(hostedZoneID string) (rrsets []*route53.ResourceRecordSet, err error) {
  input := route53.ListResourceRecordSetsInput{
    HostedZoneId: aws.String(hostedZoneID),
  }

  for {
    var resp *route53.ListResourceRecordSetsOutput
    resp, err = client.r53.ListResourceRecordSets(&input)
    if err != nil {
      return rrsets, err
    }
    rrsets = append(rrsets, resp.ResourceRecordSets...)

    if *resp.IsTruncated {
      input.StartRecordName = resp.NextRecordName
      input.StartRecordType = resp.NextRecordType
      input.StartRecordIdentifier = resp.NextRecordIdentifier
    } else {
      break
    }
  }

  return rrsets, nil
}

func compareHostedZoneName(input string, output string) bool {
  re, _ := regexp.Compile(`\.$`)
  i := re.ReplaceAllString(input, "")
  o := re.ReplaceAllString(output, "")
  return i == o
}

// GetReverseHostedZoneID ...
func GetReverseHostedZoneID(ip net.IP, rInfos ReverseHostedZoneInfos) (hostedZoneID string, err error) {
  for _, zoneInfo := range rInfos {
    ipnet := zoneInfo.Network
    if ipnet.Contains(ip) {
      return zoneInfo.HostedZoneID, nil
    }
  }
  return hostedZoneID, fmt.Errorf("not found (%s)", ip.String())
}

func (client *AWSClientImpl)AddAResourceRecordSet(ip net.IP, hostname string, hostedZoneID string, rInfos ReverseHostedZoneInfos) (err error) {
  inputForA := &route53.ChangeResourceRecordSetsInput{
    ChangeBatch: &route53.ChangeBatch{
      HostedZoneID: aws.String(hostedZoneID),
      Changes: []*route53.Change{
        Action: aws.String("CREATE"),
        ResourceRecordSet: &route53.ResourceRecordSet{
          Name: aws.String(hostname),
          ResourceRecords: []*route53.ResourceRecord{
            Value: aws.String(ip.String()),
          },
          TTL: aws.Int64(600),
          Type: aws.String("A"),
        },
      },
    },
  }
  reverseHostedZoneID, err := GetReverseHostedZoneID(ip, rInfos)
  if err != nil {
    return err
  }
  ptrRecord := GenerateReverseRecord(ip)
  inputForPTR := &route53.ChangeResourceRecordSetsInput{
    ChangeBatch: &route53.ChangeBatch{
      HostedZoneID: aws.String(reverseHostedZoneID),
      Changes: []*route53.Change{
        Action: aws.String("CREATE"),
        ResourceRecordSet: &route53.ResourceRecordSet{
          Name: aws.String(ptrRecord),
          ResourceRecords: []*route53.ResourceRecord{
            Value: aws.String(hostname),
          },
          TTL: aws.Int64(600),
          Type: aws.String("PTR"),
        },
      },
    },
  }
  resp, err := client.r53.ChangeResourceRecordSets(inputForA)
  if err != nil {
    return err
  }
}