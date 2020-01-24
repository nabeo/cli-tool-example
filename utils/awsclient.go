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
  WaitUntilResourceRecordSetsChanged(input *route53.GetChangeInput) error
}

// ReverseHostedZoneInfos ...
type ReverseHostedZoneInfos struct {
  ReverseHostedZoneInfo []ReverseHostedZoneInfo
}

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

// CreateReverseHostedZoneInfo ...
func (client *AWSClientImpl) CreateReverseHostedZoneInfo(networkCIDR string, zoneName string) (rInfo ReverseHostedZoneInfo, err error) {
  rInfo.NetworkCIDR = networkCIDR
  rInfo.HostedZoneName = zoneName

  _, rInfo.Network, err = net.ParseCIDR(networkCIDR)
  if err != nil {
    return rInfo, err
  }

  rInfo.HostedZoneID, err = client.GetHostedZoneID(zoneName)
  if err != nil {
    return rInfo, err
  }

  return rInfo, nil
}

// GetReverseHostedZoneID ...
func GetReverseHostedZoneID(ip net.IP, rInfos ReverseHostedZoneInfos) (hostedZoneID string, err error) {
  for _, zoneInfo := range rInfos.ReverseHostedZoneInfo {
    ipnet := zoneInfo.Network
    if ipnet.Contains(ip) {
      return zoneInfo.HostedZoneID, nil
    }
  }
  return hostedZoneID, fmt.Errorf("not found (%s)", ip.String())
}

// AddAResourceRecordSet ...
func (client *AWSClientImpl) AddAResourceRecordSet(ip net.IP, hostname string, hostedZoneID string, rInfos ReverseHostedZoneInfos) (err error) {
  err = client.createAResourceRecordSet(ip, hostname, hostedZoneID)
  if err != nil {
    return err
  }

  err = client.createPtrResourceRecordSet(ip, hostname, rInfos)
  if err != nil {
    rr, e := client.GetResourceRecordSetByName(ip.String(), hostedZoneID)
    if e != nil {
      return e
    }
    rolebackErr := client.deleteAResourceRecordSet(&rr, hostedZoneID)
    if rolebackErr != nil {
      return rolebackErr
    }
    return err
  }

  return nil
}

// RemoveAResourceRecordSet ...
func (client *AWSClientImpl) RemoveAResourceRecordSet(rrset *route53.ResourceRecordSet, ip net.IP, hostname string, hostedZoneID string, rInfos ReverseHostedZoneInfos) (err error) {
  err = client.deleteAResourceRecordSet(rrset, hostedZoneID)
  if err != nil {
    return err
  }

  err = client.deletePtrResourceRecordSet(ip, rInfos)
  if err != nil {
    rolebackErr := client.createAResourceRecordSet(ip, hostname, hostedZoneID)
    if rolebackErr != nil {
      return rolebackErr
    }
    return err
  }

  return nil
}

// AddCnameResourceRecordSet ...
func (client *AWSClientImpl) AddCnameResourceRecordSet(hostname string, cnameHostname string, hostedZoneID string) (err error) {
  input := &route53.ChangeResourceRecordSetsInput{
    HostedZoneId: aws.String(hostedZoneID),
    ChangeBatch: &route53.ChangeBatch{
      Changes: []*route53.Change{
        {
          Action: aws.String(route53.ChangeActionCreate),
          ResourceRecordSet: &route53.ResourceRecordSet{
            Name: aws.String(hostname),
            ResourceRecords: []*route53.ResourceRecord{
              {
                Value: aws.String(cnameHostname),
              },
            },
            TTL:  aws.Int64(600),
            Type: aws.String(route53.RRTypeCname),
          },
        },
      },
    },
  }
  resp, err := client.r53.ChangeResourceRecordSets(input)
  if err != nil {
    return err
  }
  err = client.r53.WaitUntilResourceRecordSetsChanged(&route53.GetChangeInput{Id: resp.ChangeInfo.Id})
  if err != nil {
    return err
  }
  return nil
}

// RemoveCnameResourceRecordSet ...
func (client *AWSClientImpl) RemoveCnameResourceRecordSet(rrset *route53.ResourceRecordSet, hostedZoneID string) (err error) {
  input := &route53.ChangeResourceRecordSetsInput{
    HostedZoneId: aws.String(hostedZoneID),
    ChangeBatch: &route53.ChangeBatch{
      Changes: []*route53.Change{
        {
          Action: aws.String(route53.ChangeActionDelete),
          ResourceRecordSet: rrset,
        },
      },
    },
  }
  return client.changeAndWaitResourceRecordSet(input)
}

func (client *AWSClientImpl) createAResourceRecordSet(ip net.IP, hostname string, hostedZoneID string) (err error) {
  inputForA := &route53.ChangeResourceRecordSetsInput{
    HostedZoneId: aws.String(hostedZoneID),
    ChangeBatch: &route53.ChangeBatch{
      Changes: []*route53.Change{
        {
          Action: aws.String(route53.ChangeActionCreate),
          ResourceRecordSet: &route53.ResourceRecordSet{
            Name: aws.String(hostname),
            ResourceRecords: []*route53.ResourceRecord{
              {
                Value: aws.String(ip.String()),
              },
            },
            TTL:  aws.Int64(600),
            Type: aws.String(route53.RRTypeA),
          },
        },
      },
    },
  }
  return client.changeAndWaitResourceRecordSet(inputForA)
}

func (client *AWSClientImpl) changeAndWaitResourceRecordSet(input *route53.ChangeResourceRecordSetsInput) (err error) {
  resp, err := client.r53.ChangeResourceRecordSets(input)
  if err != nil {
    return err
  }
  err = client.r53.WaitUntilResourceRecordSetsChanged(&route53.GetChangeInput{Id: resp.ChangeInfo.Id})
  if err != nil {
    return err
  }
  return nil
}

func (client *AWSClientImpl) deleteResourceRecordSet(rrset *route53.ResourceRecordSet, hostedZoneName string) (err error) {
  hostedZoneID, err := client.GetHostedZoneID(hostedZoneName)
  if err != nil {
    return err
  }
  input := &route53.ChangeResourceRecordSetsInput{
    HostedZoneId: aws.String(hostedZoneID),
    ChangeBatch: &route53.ChangeBatch{
      Changes: []*route53.Change{
        {
          Action: aws.String(route53.ChangeActionDelete),
          ResourceRecordSet: rrset,
        },
      },
    },
  }
  return client.changeAndWaitResourceRecordSet(input)
}

func (client *AWSClientImpl) createPtrResourceRecordSet(ip net.IP, hostname string, rInfos ReverseHostedZoneInfos) (err error) {
  reverseHostedZoneID, err := GetReverseHostedZoneID(ip, rInfos)
  if err != nil {
    return err
  }
  ptrRecord := GenerateReverseRecord(ip)
  inputForPTR := &route53.ChangeResourceRecordSetsInput{
    HostedZoneId: aws.String(reverseHostedZoneID),
    ChangeBatch: &route53.ChangeBatch{
      Changes: []*route53.Change{
        {
          Action: aws.String(route53.ChangeActionCreate),
          ResourceRecordSet: &route53.ResourceRecordSet{
            Name: aws.String(ptrRecord),
            ResourceRecords: []*route53.ResourceRecord{
              {
                Value: aws.String(hostname),
              },
            },
            TTL: aws.Int64(600),
            Type: aws.String(route53.RRTypePtr),
          },
        },
      },
    },
  }
  resp, err := client.r53.ChangeResourceRecordSets(inputForPTR)
  if err != nil {
    return err
  }
  err = client.r53.WaitUntilResourceRecordSetsChanged(&route53.GetChangeInput{Id: resp.ChangeInfo.Id})
  if err != nil {
    return err
  }
  return nil
}

func (client *AWSClientImpl) deleteAResourceRecordSet(rrset *route53.ResourceRecordSet, hostedZoneID string) (err error) {
  input := &route53.ChangeResourceRecordSetsInput{
    HostedZoneId: aws.String(hostedZoneID),
    ChangeBatch: &route53.ChangeBatch{
      Changes: []*route53.Change{
        {
          Action: aws.String(route53.ChangeActionDelete),
          ResourceRecordSet: rrset,
        },
      },
    },
  }
  return client.changeAndWaitResourceRecordSet(input)
}

func (client *AWSClientImpl) deletePtrResourceRecordSet(ip net.IP, rInfos ReverseHostedZoneInfos) (err error) {
  reverseHostedZoneID, err := GetReverseHostedZoneID(ip, rInfos)
  if err != nil {
    return err
  }
  ptrRecord := GenerateReverseRecord(ip)
  rr, err := client.GetResourceRecordSetByName(ptrRecord, reverseHostedZoneID)
  if err != nil {
    return err
  }
  input := &route53.ChangeResourceRecordSetsInput{
    HostedZoneId: aws.String(reverseHostedZoneID),
    ChangeBatch: &route53.ChangeBatch{
      Changes: []*route53.Change{
        {
          Action: aws.String(route53.ChangeActionDelete),
          ResourceRecordSet: &rr,
        },
      },
    },
  }
  return client.changeAndWaitResourceRecordSet(input)
}

// GetResourceRecordSetByName ...
func (client *AWSClientImpl) GetResourceRecordSetByName(hostname string, hostedZoneID string) (rr route53.ResourceRecordSet, err error) {
  input := &route53.ListResourceRecordSetsInput{
    HostedZoneId: aws.String(hostedZoneID),
    MaxItems: aws.String("1"),
    StartRecordName: aws.String(hostname),
  }
  resp, err := client.r53.ListResourceRecordSets(input)
  if err != nil {
    return rr, err
  }
  if *resp.IsTruncated {
    return rr, fmt.Errorf("unexpected response: response is truncated")
  }
  if hostname != *resp.ResourceRecordSets[0].Name {
    return rr, fmt.Errorf("hostname mismatch: input %s, response %s", hostname, *resp.ResourceRecordSets[0].Name)
  }
  return *resp.ResourceRecordSets[0], nil
}
