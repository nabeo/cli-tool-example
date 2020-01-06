package list

import (
	"fmt"

	"github.com/nabeo/cli-tool-example/utils"

	"github.com/urfave/cli/v2"
)

// Command ...
var Command = cli.Command{
  Name: "list",
  Aliases: []string{"l"},
  Usage: "list command",
  Action: doList,
  Flags: []cli.Flag{
    &cli.StringFlag{
      Name: "zone",
      Usage: "zone name",
      Required: true,
      Aliases: []string{"z"},
    },
  },
}

func doList(c *cli.Context) (err error) {
  zonename := c.String("zone")
  awsClient, err := utils.NewAWSClient(c)
  if err != nil {
    return err
  }

  id, err := awsClient.GetHostedZoneID(zonename)
  if err != nil {
    return err
  }

  rrsets, err := awsClient.ListAllResourceRecords(id)
  if err != nil {
    return err
  }

  for _, rrset := range rrsets {
    fmt.Printf("%s\t%s", *rrset.Type, *rrset.Name)
    for _, rr := range rrset.ResourceRecords {
      fmt.Printf("\t%s",*rr.Value)
    }
    fmt.Printf("\n")
  }
  return nil
}
