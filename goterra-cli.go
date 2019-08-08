package main

import (
	"flag"
	"fmt"
	"os"

	terraApi "github.com/osallou/goterra-cli/lib/api"
	terraModel "github.com/osallou/goterra-lib/lib/model"
)

var Version string

// ShowUsage display base options
func ShowUsage(options []string) {
	fmt.Println("Usage:")
	fmt.Println("  Expected environment variables:")
	fmt.Println("    * GOT_APIKEY: user api key")
	fmt.Println("    * GOT_URL: URL to goterra")
	for _, option := range options {
		fmt.Printf("  goterra-cli %s -h\n", option)
	}
}

func handleEditNamespace(options terraApi.OptionsDef, nsID string, args []string) error {
	cmdOptions := flag.NewFlagSet("edit options", flag.ExitOnError)
	addOwner := cmdOptions.String("add-owner", "", "Add an owner")
	addMember := cmdOptions.String("add-member", "", "Add a member")
	removeOwner := cmdOptions.String("remove-owner", "", "Remove an owner")
	removeMember := cmdOptions.String("remove-member", "", "Remove a member")
	if len(args) == 0 {
		cmdOptions.PrintDefaults()
		return nil
	}
	cmdOptions.Parse(args)

	ns, err := terraApi.GetNamespace(options, nsID)
	if err != nil {
		return err
	}

	if *addOwner != "" {
		ns.Owners = terraApi.AddToList(ns.Owners, *addOwner)
	}
	if *addMember != "" {
		ns.Members = terraApi.AddToList(ns.Members, *addMember)
	}
	if *removeOwner != "" {
		ns.Owners = terraApi.RemoveFromList(ns.Owners, *removeOwner)
	}
	if *removeMember != "" {
		ns.Members = terraApi.RemoveFromList(ns.Members, *removeMember)
	}
	err = terraApi.UpdateNamespace(options, ns)
	if err != nil {
		return err
	}
	fmt.Println("Namespace updated!")
	return nil
}

func handleNamespace(options terraApi.OptionsDef, args []string) error {
	var err error

	switch args[0] {
	case "list":
		cmdOptions := flag.NewFlagSet("edit options", flag.ExitOnError)
		showAll := cmdOptions.Bool("all", false, "Get all namespaces [admin]")
		cmdOptions.Parse(args[1:])
		err = terraApi.ListNamespaces(options, *showAll)
		break
	case "show":
		if len(args) == 1 {
			return fmt.Errorf("missing namespace id")
		}
		err = terraApi.ShowNamespace(options, args[1])
		break
	case "edit":
		if len(args) == 1 {
			return fmt.Errorf("missing namespace id")
		}
		err = handleEditNamespace(options, args[1], args[2:])
		break
	case "create":
		if len(args) == 1 {
			return fmt.Errorf("missing namespace name")
		}
		ns := terraModel.NSData{
			Name: args[1],
		}

		terraApi.CreateNamespace(options, &ns)
	}
	return err
}

func handleEndpoint(options terraApi.OptionsDef, args []string) error {
	var err error

	switch args[0] {
	case "list":
		cmdOptions := flag.NewFlagSet("edit options", flag.ExitOnError)
		nsID := cmdOptions.String("ns", "", "namespace id")
		cmdOptions.Parse(args[1:])
		err = terraApi.ListEndpoints(options, *nsID)
		break
	case "show":
		if len(args) == 1 {
			return fmt.Errorf("missing endpoint id")
		}
		cmdOptions := flag.NewFlagSet("edit options", flag.ExitOnError)
		nsID := cmdOptions.String("ns", "", "namespace id")
		epID := cmdOptions.String("id", "", "endpoint id")
		cmdOptions.Parse(args[1:])
		err = terraApi.ShowEndpoint(options, *nsID, *epID)
		break
	}
	return err
}

func handleUser(options terraApi.OptionsDef, args []string) error {
	var err error

	switch args[0] {
	case "list":
		err = terraApi.ListUsers(options)
		break
	case "show":
		if len(args) == 1 {
			return fmt.Errorf("missing user id")
		}
		err = terraApi.ShowUser(options, args[1])
		break
	}
	return err
}

type nsData terraModel.NSData

func cliUsage() {
	flag.PrintDefaults()
	fmt.Printf("Subcommands:\n")
	fmt.Printf(" * namespace\n")
	fmt.Printf(" * endpoint\n")
	fmt.Printf(" * user\n")
}

func nsUsage() {
	fmt.Println("Namespace sub commands:")
	fmt.Println(" * list: list user namespaces")
	fmt.Println(" * show NSID: show namespace NSID in details")
	fmt.Println(" * edit NSID: update namespace NSID info, see -h")
	fmt.Println(" * create NSNAME: creates a new user namespace")
}

func endpointUsage() {
	fmt.Println("Endpoint sub commands:")
	fmt.Println(" * list: list endpoints")
}

func userUsage() {
	fmt.Println("User sub commands:")
	fmt.Println(" * list: list users [admin]")
}

func main() {

	options := terraApi.OptionsDef{
		APIKEY: os.Getenv("GOT_APIKEY"),
		URL:    os.Getenv("GOT_URL"),
	}

	// mainOptions := []string{"namespace", "run"}

	var apiKey string
	var url string
	var showVersion bool

	flag.StringVar(&apiKey, "apikey", "", "Authentication API Key")
	flag.StringVar(&url, "url", "https://goterra.genouest.org", "URL to Goterra host")
	flag.BoolVar(&showVersion, "version", false, "show client version")
	flag.Usage = cliUsage
	flag.Parse()

	if showVersion {
		fmt.Printf("Version: %s\n", Version)
		os.Exit(0)
	}

	if apiKey != "" {
		options.APIKEY = apiKey
	}
	if url != "" {
		options.URL = url
	}

	if options.URL == "" || options.APIKEY == "" {
		fmt.Println("apikey and url options must not be empty")
		os.Exit(1)
	}

	args := flag.Args()

	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	token, loginErr := terraApi.Login(options.APIKEY, options.URL)
	if loginErr != nil {
		fmt.Printf("Error: %s\n", loginErr)
		os.Exit(1)
	}
	options.Token = token
	// fmt.Printf("token %s", token)

	var err error

	switch args[0] {
	case "namespace":
		if len(args) == 1 {
			nsUsage()
			os.Exit(1)
		}
		err = handleNamespace(options, args[1:])
		break
	case "endpoint":
		if len(args) == 1 {
			endpointUsage()
			os.Exit(1)
		}
		err = handleEndpoint(options, args[1:])
		break
	case "user":
		if len(args) == 1 {
			userUsage()
			os.Exit(1)
		}
		err = handleUser(options, args[1:])
		break
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}

	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
	//jsonOut, _ := json.MarshalIndent(result, "", "\t")

	//fmt.Printf("%s\n", jsonOut)

}
