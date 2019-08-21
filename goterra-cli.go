package main

import (
	"flag"
	"fmt"
	"os"

	terraApi "github.com/osallou/goterra-cli/lib/api"
	terraModel "github.com/osallou/goterra-lib/lib/model"
)

// Version is client version
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
	freeze := cmdOptions.Bool("freeze", false, "freeze namespace")
	unfreeze := cmdOptions.Bool("unfreeze", false, "unfreeze namespace")
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
	if *freeze == true {
		ns.Freeze = true
	}
	if *unfreeze == true {
		ns.Freeze = false
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
		cmdOptions := flag.NewFlagSet("list options", flag.ExitOnError)
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
		break
	case "delete":
		if len(args) == 1 {
			return fmt.Errorf("missing namespace name")
		}
		confirm := promptConfirm("Please confirm deletion")
		if confirm {
			terraApi.DeleteNamespace(options, args[1])
		}
		break
	}
	return err
}

func handleEndpoint(options terraApi.OptionsDef, args []string) error {
	var err error

	switch args[0] {
	case "list":
		cmdOptions := flag.NewFlagSet("list options", flag.ExitOnError)
		nsID := cmdOptions.String("ns", "", "namespace id")
		cmdOptions.Parse(args[1:])
		err = terraApi.ListEndpoints(options, *nsID)
		break
	case "show":

		cmdOptions := flag.NewFlagSet("show options", flag.ExitOnError)
		nsID := cmdOptions.String("ns", "", "namespace id")
		epID := cmdOptions.String("id", "", "endpoint id")
		cmdOptions.Parse(args[1:])
		if *nsID == "" && *epID == "" {
			return fmt.Errorf("missing endpoint or namespace id")
		}
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
	case "password":
		if len(args) != 3 {
			return fmt.Errorf("missing user id or password, usage: goterra user password USERID NEWPASSWORD")
		}
		err = terraApi.SetUserPassword(options, args[1], args[2])
		if err != nil {
			fmt.Printf("Password updated for user %s\n", args[1])
		}
		break
	}

	return err
}

func handleRecipe(options terraApi.OptionsDef, args []string) error {
	var err error

	switch args[0] {
	case "list":
		cmdOptions := flag.NewFlagSet("list options", flag.ExitOnError)
		nsID := cmdOptions.String("ns", "", "namespace id")
		cmdOptions.Parse(args[1:])
		err = terraApi.ListRecipes(options, *nsID)
		break
	case "show":
		cmdOptions := flag.NewFlagSet("list options", flag.ExitOnError)
		nsID := cmdOptions.String("ns", "", "namespace id")
		recipeID := cmdOptions.String("id", "", "recipe id")
		cmdOptions.Parse(args[1:])
		if *nsID == "" && *recipeID == "" {
			return fmt.Errorf("missing recipe or namespace id")
		}
		err = terraApi.ShowRecipe(options, *nsID, *recipeID)
		break
	}
	return err
}

func handleTemplate(options terraApi.OptionsDef, args []string) error {
	var err error

	switch args[0] {
	case "list":
		cmdOptions := flag.NewFlagSet("list options", flag.ExitOnError)
		nsID := cmdOptions.String("ns", "", "namespace id")
		cmdOptions.Parse(args[1:])
		err = terraApi.ListTemplates(options, *nsID)
		break
	case "show":
		cmdOptions := flag.NewFlagSet("list options", flag.ExitOnError)
		nsID := cmdOptions.String("ns", "", "namespace id")
		templateID := cmdOptions.String("id", "", "template id")
		cmdOptions.Parse(args[1:])

		if *nsID == "" && *templateID == "" {
			return fmt.Errorf("missing template or namespace id")
		}
		err = terraApi.ShowTemplate(options, *nsID, *templateID)
		break
	}
	return err
}

func handleApp(options terraApi.OptionsDef, args []string) error {
	var err error

	switch args[0] {
	case "list":
		cmdOptions := flag.NewFlagSet("list options", flag.ExitOnError)
		nsID := cmdOptions.String("ns", "", "namespace id")
		cmdOptions.Parse(args[1:])
		err = terraApi.ListApps(options, *nsID)
		break
	case "show":
		cmdOptions := flag.NewFlagSet("list options", flag.ExitOnError)
		nsID := cmdOptions.String("ns", "", "namespace id")
		appID := cmdOptions.String("id", "", "application id")
		cmdOptions.Parse(args[1:])

		if *nsID == "" && *appID == "" {
			return fmt.Errorf("missing app or namespace id")
		}
		err = terraApi.ShowApp(options, *nsID, *appID)
		break
	}
	return err
}

func handleRun(options terraApi.OptionsDef, args []string) error {
	var err error

	switch args[0] {
	case "list":
		cmdOptions := flag.NewFlagSet("list options", flag.ExitOnError)
		nsID := cmdOptions.String("ns", "", "namespace id")
		cmdOptions.Parse(args[1:])
		err = terraApi.ListRuns(options, *nsID)
		break
	case "show":
		cmdOptions := flag.NewFlagSet("list options", flag.ExitOnError)
		nsID := cmdOptions.String("ns", "", "namespace id")
		runID := cmdOptions.String("id", "", "run id")
		store := cmdOptions.Bool("store", false, "show store details (if deployed)")
		cmdOptions.Parse(args[1:])

		if *nsID == "" && *runID == "" {
			return fmt.Errorf("missing run or namespace id")
		}
		err = terraApi.ShowRun(options, *nsID, *runID, *store)
		break
	case "delete":
		cmdOptions := flag.NewFlagSet("list options", flag.ExitOnError)
		nsID := cmdOptions.String("ns", "", "namespace id")
		runID := cmdOptions.String("id", "", "run id")
		cmdOptions.Parse(args[1:])
		if *nsID == "" && *runID == "" {
			return fmt.Errorf("missing run or namespace id")
		}
		confirm := promptConfirm("Please confirm deletion")
		if confirm {
			terraApi.DeleteRun(options, *nsID, *runID)
		}
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
	fmt.Printf(" * recipe\n")
	fmt.Printf(" * template\n")
	fmt.Printf(" * app\n")
	fmt.Printf(" * user\n")
	fmt.Printf(" * run\n")
}

func nsUsage() {
	fmt.Println("Namespace sub commands:")
	fmt.Println(" * list: list user namespaces")
	fmt.Println(" * show NSID: show namespace NSID in details")
	fmt.Println(" * edit NSID: update namespace NSID info, see -h")
	fmt.Println(" * create NSNAME: creates a new user namespace")
	fmt.Println(" * delete NSID: removes namespace")
}

func endpointUsage() {
	fmt.Println("Endpoint sub commands:")
	fmt.Println(" * list: list endpoints")
}

func userUsage() {
	fmt.Println("User sub commands:")
	fmt.Println(" * list: list users [admin]")
	fmt.Println(" * show UID: show user info [user or admin]")
	fmt.Println(" * password UID XXX: modify user password [user or admin]")
}

func recipeUsage() {
	fmt.Println("User sub commands:")
	fmt.Println(" * list: list recipes")
	fmt.Println(" * show ID: show recipe info ")
}

func templateUsage() {
	fmt.Println("User sub commands:")
	fmt.Println(" * list: list templates")
	fmt.Println(" * show ID: show template info ")
}

func appUsage() {
	fmt.Println("User sub commands:")
	fmt.Println(" * list: list applications")
	fmt.Println(" * show ID: show application info ")
}

func runUsage() {
	fmt.Println("User sub commands:")
	fmt.Println(" * list: list runs")
	fmt.Println(" * show ID: show run info ")
	fmt.Println(" * delete ID: ask to stop run ")
}

func promptConfirm(question string) bool {
	fmt.Print(question + "[y/n]:")
	var input string
	fmt.Scanln(&input)
	if input == "y" {
		return true
	}
	return false
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
	case "recipe":
		if len(args) == 1 {
			recipeUsage()
			os.Exit(1)
		}
		err = handleRecipe(options, args[1:])
		break
	case "template":
		if len(args) == 1 {
			templateUsage()
			os.Exit(1)
		}
		err = handleTemplate(options, args[1:])
		break
	case "app":
		if len(args) == 1 {
			appUsage()
			os.Exit(1)
		}
		err = handleApp(options, args[1:])
		break
	case "run":
		if len(args) == 1 {
			runUsage()
			os.Exit(1)
		}
		err = handleRun(options, args[1:])
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
