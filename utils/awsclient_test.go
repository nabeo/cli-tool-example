package utils


import (
  "testing"
  "errors"
  "net"
  "time"

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

  ChangeResourceRecordSetsInput *route53.ChangeResourceRecordSetsInput
  ChangeResourceRecordSetsOutput *route53.ChangeResourceRecordSetsOutput
  ChangeResourceRecordSetsError error

  GetChangeInput *route53.GetChangeInput

  WaitUntilResourceRecordSetsChangedError error
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
  validateError := input.Validate()
  if validateError != nil {
    c.t.Errorf("validate error: %s", validateError.Error())
  }

  return c.listResourceRecordSetsOutput, c.listHostedZonesByNameError
}

func (c DummyRoute53Client) ChangeResourceRecordSets(input *route53.ChangeResourceRecordSetsInput) (*route53.ChangeResourceRecordSetsOutput, error) {
  expectedInput := awsutil.StringValue(c.ChangeResourceRecordSetsInput)
  actualInput := awsutil.StringValue(input)
  if expectedInput != actualInput {
    c.t.Errorf("unexpected input: expected %v, actual %v", expectedInput, actualInput)
  }
  validateError := input.Validate()
  if validateError != nil {
    c.t.Errorf("validate error: %s", validateError.Error())
  }

  return c.ChangeResourceRecordSetsOutput, c.ChangeResourceRecordSetsError
}

func (c DummyRoute53Client) WaitUntilResourceRecordSetsChanged(input *route53.GetChangeInput) error {
  expectedInput := awsutil.StringValue(c.GetChangeInput)
  actualInput := awsutil.StringValue(input)
  if expectedInput != actualInput {
    c.t.Errorf("unexpected input: expected %v, actual %v", expectedInput, actualInput)
  }
  validateError := input.Validate()
  if validateError != nil {
    c.t.Errorf("validate error: %v", validateError.Error())
  }

  return c.WaitUntilResourceRecordSetsChangedError
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
      HostedZoneName: "168.192.in-addr.arpa.",
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

func TestCreateAResourceRecordSet(t *testing.T) {
  patterns := []struct{
    ip net.IP
    hostname string
    hostedZoneID string

    expectedError error
  }{
    {
      ip: net.ParseIP("10.0.5.10"),
      hostname: "host.example.com",
      hostedZoneID: "ABC123",
      expectedError: nil,
    },
  }

  for idx, p := range patterns {
    awsClient := &AWSClientImpl{
      r53: &DummyRoute53Client{
        t: t,

        ChangeResourceRecordSetsInput: &route53.ChangeResourceRecordSetsInput{
          HostedZoneId: aws.String(p.hostedZoneID),
          ChangeBatch: &route53.ChangeBatch{
            Changes: []*route53.Change{
              {
                Action: aws.String(route53.ChangeActionCreate),
                ResourceRecordSet: &route53.ResourceRecordSet{
                  Name: aws.String(p.hostname),
                  ResourceRecords: []*route53.ResourceRecord{
                    {
                      Value: aws.String(p.ip.String()),
                    },
                  },
                  TTL:  aws.Int64(600),
                  Type: aws.String("A"),
                },
              },
            },
          },
        },
        ChangeResourceRecordSetsOutput: &route53.ChangeResourceRecordSetsOutput{
          ChangeInfo: &route53.ChangeInfo{
            Comment: aws.String("dummy comment"),
            Id: aws.String("XYZ789"),
            Status: aws.String(route53.ChangeStatusInsync),
            SubmittedAt: aws.Time(time.Date(2020, 1, 13, 0, 0, 0, 0, time.UTC)),
          },
        },
        ChangeResourceRecordSetsError: nil,

        GetChangeInput: &route53.GetChangeInput{
          Id: aws.String("XYZ789"),
        },
      },
    }
    err := awsClient.createAResourceRecordSet(p.ip, p.hostname, p.hostedZoneID)
    if err != nil && err.Error() != p.expectedError.Error() {
      t.Errorf("unexpected error (%d): expected error %v, actual error %v", idx, p.expectedError, err)
    }
  }
}

func TestDeleteAResourceRecordSet(t *testing.T) {
  patterns := []struct{
    ip net.IP
    hostname string
    hostedZoneID string

    expectedError error
  }{
    {
      ip: net.ParseIP("10.0.5.10"),
      hostname: "host.example.com",
      hostedZoneID: "ABC123",
      expectedError: nil,
    },
  }

  for idx, p := range patterns {
    awsClient := &AWSClientImpl{
      r53: &DummyRoute53Client{
        t: t,

        ChangeResourceRecordSetsInput: &route53.ChangeResourceRecordSetsInput{
          HostedZoneId: aws.String(p.hostedZoneID),
          ChangeBatch: &route53.ChangeBatch{
            Changes: []*route53.Change{
              {
                Action: aws.String(route53.ChangeActionDelete),
                ResourceRecordSet: &route53.ResourceRecordSet{
                  Name: aws.String(p.hostname),
                  ResourceRecords: []*route53.ResourceRecord{
                    {
                      Value: aws.String(p.ip.String()),
                    },
                  },
                  TTL:  aws.Int64(600),
                  Type: aws.String("A"),
                },
              },
            },
          },
        },
        ChangeResourceRecordSetsOutput: &route53.ChangeResourceRecordSetsOutput{
          ChangeInfo: &route53.ChangeInfo{
            Comment: aws.String("dummy comment"),
            Id: aws.String("XYZ789"),
            Status: aws.String(route53.ChangeStatusInsync),
            SubmittedAt: aws.Time(time.Date(2020, 1, 13, 0, 0, 0, 0, time.UTC)),
          },
        },
        ChangeResourceRecordSetsError: nil,

        GetChangeInput: &route53.GetChangeInput{
          Id: aws.String("XYZ789"),
        },
      },
    }
    err := awsClient.deleteAResourceRecordSet(p.ip, p.hostname, p.hostedZoneID)
    if err != nil && err.Error() != p.expectedError.Error() {
      t.Errorf("unexpected error (%d): expected error %v, actual error %v", idx, p.expectedError, err)
    }
  }
}

func TestCreatePtrResourceRecordSet(t *testing.T) {
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
    hostname string
    hostedZoneID string

    expectedError error
  }{
    {
      ip: net.ParseIP("10.0.5.10"),
      hostname: "host.example.com",
      expectedError: nil,
    },
  }
  for idx, p := range patterns {
    awsClient := &AWSClientImpl{
      r53: &DummyRoute53Client{
        t: t,

        ChangeResourceRecordSetsInput: &route53.ChangeResourceRecordSetsInput{
          HostedZoneId: aws.String("ABC123"),
          ChangeBatch: &route53.ChangeBatch{
            Changes: []*route53.Change{
              {
                Action: aws.String(route53.ChangeActionCreate),
                ResourceRecordSet: &route53.ResourceRecordSet{
                  Name: aws.String("10.5.0.10.in-addr.arpa."),
                  ResourceRecords: []*route53.ResourceRecord{
                    {
                      Value: aws.String(p.hostname),
                    },
                  },
                  TTL: aws.Int64(600),
                  Type: aws.String("PTR"),
                },
              },
            },
          },
        },
        ChangeResourceRecordSetsOutput: &route53.ChangeResourceRecordSetsOutput{
           ChangeInfo: &route53.ChangeInfo{
            Comment: aws.String("dummy comment"),
            Id: aws.String("XYZ789"),
            Status: aws.String(route53.ChangeStatusInsync),
            SubmittedAt: aws.Time(time.Date(2020, 1, 13, 0, 0, 0, 0, time.UTC)),
          },
        },
        ChangeResourceRecordSetsError: nil,

        GetChangeInput: &route53.GetChangeInput{
          Id: aws.String("XYZ789"),
        },
      },
    }
    err := awsClient.createPtrResourceRecordSet(p.ip, p.hostname, rInfos)
    if err != nil && err.Error() != p.expectedError.Error() {
      t.Errorf("unexpected error (%d): expected error %v, actual error %v", idx, p.expectedError, err)
    }
  }
}

func TestDeletePtrResourceRecordSet(t *testing.T) {
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
    hostname string
    hostedZoneID string

    expectedError error
  }{
    {
      ip: net.ParseIP("10.0.5.10"),
      hostname: "host.example.com",
      expectedError: nil,
    },
  }
  for idx, p := range patterns {
    awsClient := &AWSClientImpl{
      r53: &DummyRoute53Client{
        t: t,

        ChangeResourceRecordSetsInput: &route53.ChangeResourceRecordSetsInput{
          HostedZoneId: aws.String("ABC123"),
          ChangeBatch: &route53.ChangeBatch{
            Changes: []*route53.Change{
              {
                Action: aws.String(route53.ChangeActionDelete),
                ResourceRecordSet: &route53.ResourceRecordSet{
                  Name: aws.String("10.5.0.10.in-addr.arpa."),
                  ResourceRecords: []*route53.ResourceRecord{
                    {
                      Value: aws.String(p.hostname),
                    },
                  },
                  TTL: aws.Int64(600),
                  Type: aws.String("PTR"),
                },
              },
            },
          },
        },
        ChangeResourceRecordSetsOutput: &route53.ChangeResourceRecordSetsOutput{
           ChangeInfo: &route53.ChangeInfo{
            Comment: aws.String("dummy comment"),
            Id: aws.String("XYZ789"),
            Status: aws.String(route53.ChangeStatusInsync),
            SubmittedAt: aws.Time(time.Date(2020, 1, 13, 0, 0, 0, 0, time.UTC)),
          },
        },
        ChangeResourceRecordSetsError: nil,

        GetChangeInput: &route53.GetChangeInput{
          Id: aws.String("XYZ789"),
        },
      },
    }
    err := awsClient.deletePtrResourceRecordSet(p.ip, p.hostname, rInfos)
    if err != nil && err.Error() != p.expectedError.Error() {
      t.Errorf("unexpected error (%d): expected error %v, actual error %v", idx, p.expectedError, err)
    }
  }
}

func TestAddCnameResourceRecordSet(t *testing.T) {
  patterns := []struct{
    hostname string
    cnameHostname string
    hostedZoneID string

    expectedError error
  }{
    {
      hostname: "www.example.com",
      cnameHostname: "cname.example.com",
      hostedZoneID: "ABC123",
      expectedError: nil,
    },
  }

  for idx, p := range patterns {
    awsClient := &AWSClientImpl{
      r53: &DummyRoute53Client{
        t: t,

        ChangeResourceRecordSetsInput: &route53.ChangeResourceRecordSetsInput{
          HostedZoneId: aws.String(p.hostedZoneID),
          ChangeBatch: &route53.ChangeBatch{
            Changes: []*route53.Change{
              {
                Action: aws.String(route53.ChangeActionCreate),
                ResourceRecordSet: &route53.ResourceRecordSet{
                  Name: aws.String(p.hostname),
                  ResourceRecords: []*route53.ResourceRecord{
                    {
                      Value: aws.String(p.cnameHostname),
                    },
                  },
                  TTL: aws.Int64(600),
                  Type: aws.String("CNAME"),
                },
              },
            },
          },
        },
        ChangeResourceRecordSetsOutput: &route53.ChangeResourceRecordSetsOutput{
           ChangeInfo: &route53.ChangeInfo{
            Comment: aws.String("dummy comment"),
            Id: aws.String("XYZ789"),
            Status: aws.String(route53.ChangeStatusInsync),
            SubmittedAt: aws.Time(time.Date(2020, 1, 13, 0, 0, 0, 0, time.UTC)),
          },
        },
        ChangeResourceRecordSetsError: nil,

        GetChangeInput: &route53.GetChangeInput{
          Id: aws.String("XYZ789"),
        },
      },
    }
    err := awsClient.AddCnameResourceRecordSet(p.hostname, p.cnameHostname, p.hostedZoneID)
    if err != nil && err.Error() != p.expectedError.Error() {
      t.Errorf("unexpected error (%d): expected error %v, actual error %v", idx, p.expectedError, err)
    }
  }
}

func TestRemoveCnameResourceRecordSet(t *testing.T) {
  patterns := []struct{
    hostname string
    cnameHostname string
    hostedZoneID string

    expectedError error
  }{
    {
      hostname: "www.example.com",
      cnameHostname: "cname.example.com",
      hostedZoneID: "ABC123",
      expectedError: nil,
    },
  }

  for idx, p := range patterns {
    awsClient := &AWSClientImpl{
      r53: &DummyRoute53Client{
        t: t,

        ChangeResourceRecordSetsInput: &route53.ChangeResourceRecordSetsInput{
          HostedZoneId: aws.String(p.hostedZoneID),
          ChangeBatch: &route53.ChangeBatch{
            Changes: []*route53.Change{
              {
                Action: aws.String(route53.ChangeActionDelete),
                ResourceRecordSet: &route53.ResourceRecordSet{
                  Name: aws.String(p.hostname),
                  ResourceRecords: []*route53.ResourceRecord{
                    {
                      Value: aws.String(p.cnameHostname),
                    },
                  },
                  TTL: aws.Int64(600),
                  Type: aws.String("CNAME"),
                },
              },
            },
          },
        },
        ChangeResourceRecordSetsOutput: &route53.ChangeResourceRecordSetsOutput{
          ChangeInfo: &route53.ChangeInfo{
            Comment: aws.String("dummy comment"),
            Id: aws.String("XYZ789"),
            Status: aws.String(route53.ChangeStatusInsync),
            SubmittedAt: aws.Time(time.Date(2020, 1, 13, 0, 0, 0, 0, time.UTC)),
          },
        },
        ChangeResourceRecordSetsError: nil,

        GetChangeInput: &route53.GetChangeInput{
          Id: aws.String("XYZ789"),
        },
      },
    }
    err := awsClient.RemoveCnameResourceRecordSet(p.hostname, p.cnameHostname, p.hostedZoneID)
    if err != nil && err.Error() != p.expectedError.Error() {
      t.Errorf("unexpected error (%d): expected error %v, actual error %v", idx, p.expectedError, err)
    }
  }
}
