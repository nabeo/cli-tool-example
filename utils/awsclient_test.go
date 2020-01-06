package utils


import (
  "testing"
  "errors"
  "net"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/service/route53"
)

type DummyRoute53Client struct {
  t *testing.T

  listHostedZonesByNameInput *route53.ListHostedZonesByNameInput
  listHostedZonesByNameOutput *route53.ListHostedZonesByNameOutput
  listHostedZonesByNameError error

  listResourceRecordSetsInput *route53.ListResourceRecordSetsInput
  listResourceRecordSetsOutput *route53.ListResourceRecordSetsOutput
  listResourceRecordSetsError error
}

func (c DummyRoute53Client) ListHostedZonesByName(input *route53.ListHostedZonesByNameInput) (*route53.ListHostedZonesByNameOutput, error) {
  expectedInput := awsutil.StringValue(c.listHostedZonesByNameInput)
  actualInput := awsutil.StringValue(input)
  if expectedInput != actualInput {
    c.t.Errorf("unexpected input: expected %v, actual %v", expectedInput, actualInput)
  }

  return c.listHostedZonesByNameOutput, c.listHostedZonesByNameError
}

func (c DummyRoute53Client) ListResourceRecordSets(input *route53.ListResourceRecordSetsInput) (*route53.ListResourceRecordSetsOutput, error) {
  expectedInput := awsutil.StringValue(c.listResourceRecordSetsInput)
  actualInput := awsutil.StringValue(input)
  if expectedInput != actualInput {
    c.t.Errorf("unexpected input: expected %v, actual %v", expectedInput, actualInput)
  }

  return c.listResourceRecordSetsOutput, c.listHostedZonesByNameError
}

func TestGetHostedZoneID(t *testing.T) {
  patterns := []struct {
    hostedZoneName string

    listHostedZonesByNameInput *route53.ListHostedZonesByNameInput
    listHostedZonesByNameOutput *route53.ListHostedZonesByNameOutput
    listHostedZonesByNameError error

    expectedHostedZoneID string
    expectedError error
  }{
    {
      hostedZoneName: "example.com.",
      
      listHostedZonesByNameInput: &route53.ListHostedZonesByNameInput{
        DNSName: aws.String("example.com."),
        MaxItems: aws.String("1"),
      },
      listHostedZonesByNameOutput: &route53.ListHostedZonesByNameOutput{
        HostedZones: []*route53.HostedZone{
          &route53.HostedZone{
            Name: aws.String("example.com."),
            Id: aws.String("/hostedzone/ABC123"),
          },
        },
      },

    expectedHostedZoneID: "ABC123",
    expectedError: nil,
    },
    {
      hostedZoneName: "notfound.com.",

      listHostedZonesByNameInput: &route53.ListHostedZonesByNameInput{
        DNSName: aws.String("notfound.com."),
        MaxItems: aws.String("1"),
      },
      listHostedZonesByNameOutput: nil,
      listHostedZonesByNameError: errors.New("error"),

      expectedHostedZoneID: "",
      expectedError: errors.New("HostedZone not found: notfound.com."),
    },
    {
      hostedZoneName: "example.com.",

      listHostedZonesByNameInput: &route53.ListHostedZonesByNameInput{
        DNSName: aws.String("example.com."),
        MaxItems: aws.String("1"),
      },
      listHostedZonesByNameOutput: &route53.ListHostedZonesByNameOutput{
        HostedZones: []*route53.HostedZone{
          &route53.HostedZone{
            Name: aws.String("example.net."),
            Id: aws.String("/hostedzone/EFG456"),
          },
        },
      },
      listHostedZonesByNameError: nil,

      expectedHostedZoneID: "",
      expectedError: errors.New("unexpected HostedZone Name: expected example.com., actual example.net."),
    },
  }

  for _, pattern := range patterns {
    awsClient := &AWSClientImpl{
      r53: &DummyRoute53Client{
        t: t,
        
        listHostedZonesByNameInput: pattern.listHostedZonesByNameInput,
        listHostedZonesByNameOutput: pattern.listHostedZonesByNameOutput,
        listHostedZonesByNameError: pattern.listHostedZonesByNameError,
      },
    }

    hostedZoneID, err := awsClient.GetHostedZoneID(pattern.hostedZoneName)
    if err != nil && err.Error() != pattern.expectedError.Error() {
      t.Errorf("unexpected Error: expected error %v, actual error %v", pattern.expectedError, err)
    } else if hostedZoneID != pattern.expectedHostedZoneID {
      t.Errorf("unexpected hostedZoneID: expected %s, actual %s", pattern.expectedHostedZoneID, hostedZoneID)
    }
  }
}

func TestCompareHostedZoneName (t *testing.T) {
  patterns := []struct{
    a string
    b string
    expected bool
  }{
    { "example.com", "example.com", true },
    { "example.com", "example.com.", true },
    { "example.com.", "example.com", true },
    { "example.com.", "example.com.", true },
    { "hoge.example.com", "example.com", false },
    { "hoge.example.com", "hoge1example.com", false },
  }
  
  for idx, pattern := range patterns {
    actual := compareHostedZoneName(pattern.a, pattern.b)
    if pattern.expected != actual {
      t.Errorf("pattern %d (%s, %s): want %t, actual %t", idx, pattern.a, pattern.b, pattern.expected, actual)
    }
  }
}

func TestGetReverseHostedZoneID(t *testing.T) {
  var rInfos ReverseHostedZoneInfos
  rInfos = []ReverseHostedZoneInfo{
    {
      Network: &net.IPNet{
        IP: net.IPv4(10,0,0,0),
        Mask: net.IPv4Mask(255,0,0,0),
      },
      NetworkCIDR: "10.0.0.0/8",
      HostedZoneID: "ABC123",
      HostedZoneName: "10.in-addr.arpa.",
    },
    {
      Network: &net.IPNet{
        IP: net.ParseIP("192.168.0.0"),
        Mask: net.IPv4Mask(255,255,0,0),
      },
      NetworkCIDR: "192.168.0.0/16",
      HostedZoneID: "EFG456",
      HostedZoneName: "10.in-addr.arpa.",
    },
  }

  patterns := []struct{
    ip net.IP
    
    expectedZoneID string
    expectedError error
  }{
    {
      ip: net.ParseIP("10.1.0.15"),
      expectedZoneID: "ABC123",
      expectedError: nil,
    },
    {
      ip: net.IPv4(10,0,0,128),
      expectedZoneID: "ABC123",
      expectedError: nil,
    },
    {
      ip: net.ParseIP("192.168.0.11"),
      expectedZoneID: "EFG456",
      expectedError: nil,
    },
    {
      ip: net.IPv4(192,168,1,15),
      expectedZoneID: "EFG456",
      expectedError: nil,
    },
    {
      ip: net.ParseIP("172.21.4.15"),
      expectedZoneID: "",
      expectedError: errors.New("not found (172.21.4.15)"),
    },
  }

  for idx, p := range patterns {
    actual, err := GetReverseHostedZoneID(p.ip, rInfos)
    if err != nil && err.Error() != p.expectedError.Error() {
      t.Errorf("unexpected error(%d): expected error %v, actual error %v", idx, p.expectedError, err)
    } else if actual != p.expectedZoneID {
      t.Errorf("unexpected HostedZone ID(%d): expected %s, actual %s", idx, p.expectedZoneID, actual)
    }
  }
}
