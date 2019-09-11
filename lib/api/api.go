package goterraapi

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"gopkg.in/yaml.v2"

	terraModel "github.com/osallou/goterra-lib/lib/model"
	terraUser "github.com/osallou/goterra-lib/lib/user"
)

// AuthData is result struct for authentication with user data and an authentication token
type AuthData struct {
	User  terraUser.User `json:"user"`
	Token string         `json:"token"`
}

// NSResp is json response to namespace list
type NSResp struct {
	NS []terraModel.NSData `json:"ns"`
}

// OptionsDef defined connection info
type OptionsDef struct {
	APIKEY string
	URL    string
	Token  string
}

// CreateUser creates a new user
func CreateUser(options OptionsDef, user *terraUser.User) error {
	data, dataErr := json.Marshal(user)
	if dataErr != nil {
		return dataErr
	}
	client := http.Client{}
	nsReq, authReqErr := http.NewRequest("POST", fmt.Sprintf("%s/auth/register", options.URL), bytes.NewBuffer(data))
	nsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", options.Token))
	nsReq.Header.Add("Content-Type", "application/json")
	if authReqErr != nil {
		return authReqErr

	}
	nsResp, nsRespErr := client.Do(nsReq)
	if nsRespErr != nil {
		return authReqErr

	}
	defer nsResp.Body.Close()
	if nsResp.StatusCode != 200 {
		var data map[string]interface{}
		json.NewDecoder(nsResp.Body).Decode(&data)
		return fmt.Errorf("Failed to create user: %s", data["message"].(string))
	}
	return nil
}

// Login authenticate users and return a token
func Login(apiKey string, url string) (string, error) {
	client := http.Client{}
	authReq, authReqErr := http.NewRequest("GET", fmt.Sprintf("%s/auth/api", url), nil)
	authReq.Header.Set("X-API-Key", apiKey)
	if authReqErr != nil {
		return "", authReqErr

	}
	authResp, authRespErr := client.Do(authReq)
	if authRespErr != nil {
		return "", authReqErr

	}
	defer authResp.Body.Close()
	if authResp.StatusCode != 200 {
		return "", fmt.Errorf("Failed to authenticate")
	}

	var authData AuthData
	json.NewDecoder(authResp.Body).Decode(&authData)
	return authData.Token, nil
}

// GetNamespaces returns user namespaces
func GetNamespaces(options OptionsDef, showAll bool) ([]terraModel.NSData, error) {
	client := http.Client{}

	nsReq, authReqErr := http.NewRequest("GET", fmt.Sprintf("%s/deploy/ns", options.URL), nil)
	nsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", options.Token))
	nsReq.Header.Add("Content-Type", "application/json")

	if showAll {
		q := nsReq.URL.Query()
		q.Add("all", "1")
		nsReq.URL.RawQuery = q.Encode()
	}
	if authReqErr != nil {
		return nil, authReqErr

	}
	nsResp, nsRespErr := client.Do(nsReq)
	if nsRespErr != nil {
		return nil, authReqErr

	}
	defer nsResp.Body.Close()
	if nsResp.StatusCode != 200 {
		var data map[string]interface{}
		json.NewDecoder(nsResp.Body).Decode(&data)
		return nil, fmt.Errorf("Failed to get namespaces: %s", data["message"].(string))
	}

	var nsResult NSResp
	json.NewDecoder(nsResp.Body).Decode(&nsResult)
	data := nsResult.NS
	return data, nil
}

// ListNamespaces list the user namespaces
func ListNamespaces(options OptionsDef, showAll bool) error {
	data, err := GetNamespaces(options, showAll)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, '\t', tabwriter.AlignRight|tabwriter.Debug)
	fmt.Fprintf(w, "%s\t%s\t%s\n", "ID", "Name", "Owners")
	for _, ns := range data {
		fmt.Fprintf(w, "%s\t%s\t%s\n", ns.ID.Hex(), ns.Name, strings.Join(ns.Owners, ","))
	}
	w.Flush()
	return nil
}

// GetNamespace returns selected namespace
func GetNamespace(options OptionsDef, nsID string) (*terraModel.NSData, error) {
	client := http.Client{}
	nsReq, authReqErr := http.NewRequest("GET", fmt.Sprintf("%s/deploy/ns/%s", options.URL, nsID), nil)
	nsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", options.Token))
	nsReq.Header.Add("Content-Type", "application/json")
	if authReqErr != nil {
		return nil, authReqErr

	}
	nsResp, nsRespErr := client.Do(nsReq)
	if nsRespErr != nil {
		return nil, authReqErr

	}
	defer nsResp.Body.Close()
	if nsResp.StatusCode != 200 {
		var data map[string]interface{}
		json.NewDecoder(nsResp.Body).Decode(&data)
		return nil, fmt.Errorf("Failed to get namespace: %s", data["message"].(string))
	}

	var nsResult map[string]terraModel.NSData
	json.NewDecoder(nsResp.Body).Decode(&nsResult)
	nsData := nsResult["ns"]
	return &nsData, nil
}

// ShowNamespace displays the user namespaces
func ShowNamespace(options OptionsDef, nsID string) error {

	data, err := GetNamespace(options, nsID)
	if err != nil {
		return err
	}
	fmt.Printf("id: %s\n", data.ID.Hex())
	yamlData, _ := yaml.Marshal(data)
	fmt.Printf("%s\n", yamlData)
	return nil
}

// AddToList adds an element to list without duplicates and returns updated list
func AddToList(members []string, newMember string) []string {
	exists := false
	for _, member := range members {
		if member == newMember {
			exists = true
			break
		}
	}
	if exists {
		return members
	}
	return append(members, newMember)
}

// RemoveFromList delete an element from list if present and returns updated list
func RemoveFromList(members []string, deprecatedMember string) []string {
	index := -1
	for pos, member := range members {
		if member == deprecatedMember {
			index = pos
			break
		}
	}
	if index == -1 {
		return members
	}
	return append(members[:index], members[index+1:]...)
}

// UpdateNamespace updates namespace data
func UpdateNamespace(options OptionsDef, ns *terraModel.NSData) error {
	data, dataErr := json.Marshal(ns)
	if dataErr != nil {
		return dataErr
	}
	client := http.Client{}
	nsReq, authReqErr := http.NewRequest("PUT", fmt.Sprintf("%s/deploy/ns/%s", options.URL, ns.ID.Hex()), bytes.NewBuffer(data))
	nsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", options.Token))
	nsReq.Header.Add("Content-Type", "application/json")
	if authReqErr != nil {
		return authReqErr

	}
	nsResp, nsRespErr := client.Do(nsReq)
	if nsRespErr != nil {
		return authReqErr

	}
	defer nsResp.Body.Close()
	if nsResp.StatusCode != 200 {
		var data map[string]interface{}
		json.NewDecoder(nsResp.Body).Decode(&data)
		return fmt.Errorf("Failed to update namespace: %s", data["message"].(string))
	}

	return nil
}

// DeleteNamespace removes namespace
func DeleteNamespace(options OptionsDef, nsID string) error {
	client := http.Client{}
	nsReq, authReqErr := http.NewRequest("DELETE", fmt.Sprintf("%s/deploy/ns/%s", options.URL, nsID), nil)
	nsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", options.Token))
	nsReq.Header.Add("Content-Type", "application/json")
	if authReqErr != nil {
		return authReqErr

	}
	nsResp, nsRespErr := client.Do(nsReq)
	if nsRespErr != nil {
		return authReqErr

	}
	defer nsResp.Body.Close()
	if nsResp.StatusCode != 200 {
		var data map[string]interface{}
		json.NewDecoder(nsResp.Body).Decode(&data)
		return fmt.Errorf("Failed to delete namespace: %s", data["message"].(string))
	}

	return nil
}

// CreateNamespace creates a new namespace
func CreateNamespace(options OptionsDef, ns *terraModel.NSData) error {
	data, dataErr := json.Marshal(ns)
	if dataErr != nil {
		return dataErr
	}
	client := http.Client{}
	nsReq, authReqErr := http.NewRequest("POST", fmt.Sprintf("%s/deploy/ns/%s", options.URL, ns.ID.Hex()), bytes.NewBuffer(data))
	nsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", options.Token))
	nsReq.Header.Add("Content-Type", "application/json")
	if authReqErr != nil {
		return authReqErr

	}
	nsResp, nsRespErr := client.Do(nsReq)
	if nsRespErr != nil {
		return authReqErr

	}
	defer nsResp.Body.Close()
	if nsResp.StatusCode != 201 {
		var data map[string]interface{}
		json.NewDecoder(nsResp.Body).Decode(&data)
		return fmt.Errorf("Failed to create namespace: %s", data["message"].(string))
	}

	return nil
}

// GetEndpoints returns endpoints for namespace or public endpoints if nsID is empty
func GetEndpoints(options OptionsDef, nsID string) ([]terraModel.EndPoint, error) {
	client := http.Client{}
	nsReq, authReqErr := http.NewRequest("GET", fmt.Sprintf("%s/deploy/endpoints", options.URL), nil)
	if nsID != "" {
		nsReq, authReqErr = http.NewRequest("GET", fmt.Sprintf("%s/deploy/ns/%s/endpoint", options.URL, nsID), nil)
	}
	nsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", options.Token))
	nsReq.Header.Add("Content-Type", "application/json")

	if authReqErr != nil {
		return nil, authReqErr

	}
	nsResp, nsRespErr := client.Do(nsReq)
	if nsRespErr != nil {
		return nil, authReqErr

	}
	defer nsResp.Body.Close()
	if nsResp.StatusCode != 200 {
		var data map[string]interface{}
		json.NewDecoder(nsResp.Body).Decode(&data)
		return nil, fmt.Errorf("Failed to get endpoints: %s", data["message"].(string))
	}

	var nsResult map[string][]terraModel.EndPoint
	json.NewDecoder(nsResp.Body).Decode(&nsResult)
	data := nsResult["endpoints"]
	return data, nil
}

// GetEndpoint returns selected endpoint
func GetEndpoint(options OptionsDef, nsID, epID string) (*terraModel.EndPoint, error) {
	client := http.Client{}
	nsReq, authReqErr := http.NewRequest("GET", fmt.Sprintf("%s/deploy/ns/%s/endpoint/%s", options.URL, nsID, epID), nil)
	nsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", options.Token))
	nsReq.Header.Add("Content-Type", "application/json")
	if authReqErr != nil {
		return nil, authReqErr

	}
	nsResp, nsRespErr := client.Do(nsReq)
	if nsRespErr != nil {
		return nil, authReqErr

	}
	defer nsResp.Body.Close()
	if nsResp.StatusCode != 200 {
		var data map[string]interface{}
		json.NewDecoder(nsResp.Body).Decode(&data)
		return nil, fmt.Errorf("Failed to get namespace: %s", data["message"].(string))
	}

	var nsResult map[string]terraModel.EndPoint
	json.NewDecoder(nsResp.Body).Decode(&nsResult)
	nsData := nsResult["endpoint"]
	return &nsData, nil
}

// ListEndpoints list the endpoints
func ListEndpoints(options OptionsDef, nsID string) error {
	data, err := GetEndpoints(options, nsID)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, '\t', tabwriter.AlignRight|tabwriter.Debug)
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", "ID", "Name", "Kind", "Public", "Namespace")
	for _, ep := range data {
		fmt.Fprintf(w, "%s\t%s\t%s\t%t\t%s\n", ep.ID.Hex(), ep.Name, ep.Kind, ep.Public, ep.Namespace)
	}
	w.Flush()
	return nil
}

// ShowEndpoint displays the endpoint
func ShowEndpoint(options OptionsDef, nsID string, epID string) error {

	data, err := GetEndpoint(options, nsID, epID)
	if err != nil {
		return err
	}
	fmt.Printf("id: %s\n", data.ID.Hex())
	yamlData, _ := yaml.Marshal(data)
	fmt.Printf("%s\n", yamlData)

	return nil
}

// GetUsers returns list of users [admin only]
func GetUsers(options OptionsDef) ([]terraUser.User, error) {
	client := http.Client{}
	nsReq, authReqErr := http.NewRequest("GET", fmt.Sprintf("%s/auth/user", options.URL), nil)
	nsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", options.Token))
	nsReq.Header.Add("Content-Type", "application/json")

	if authReqErr != nil {
		return nil, authReqErr

	}
	nsResp, nsRespErr := client.Do(nsReq)
	if nsRespErr != nil {
		return nil, authReqErr

	}
	defer nsResp.Body.Close()
	if nsResp.StatusCode != 200 {
		var data map[string]interface{}
		json.NewDecoder(nsResp.Body).Decode(&data)
		return nil, fmt.Errorf("Failed to get users: %s", data["message"].(string))
	}

	var nsResult map[string][]terraUser.User
	json.NewDecoder(nsResp.Body).Decode(&nsResult)
	data := nsResult["users"]
	return data, nil
}

// ListUsers list the users
func ListUsers(options OptionsDef) error {
	data, err := GetUsers(options)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, '\t', tabwriter.AlignRight|tabwriter.Debug)
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", "UID", "Admin", "Super user", "Email", "Kind")
	for _, user := range data {
		fmt.Fprintf(w, "%s\t%t\t%t\t%s\t%s\n", user.UID, user.Admin, user.SuperUser, user.Email, user.Kind)
	}
	w.Flush()
	return nil
}

// GetUser returns selected user
func GetUser(options OptionsDef, userID string) (*terraUser.User, error) {
	client := http.Client{}
	nsReq, authReqErr := http.NewRequest("GET", fmt.Sprintf("%s/auth/user/%s", options.URL, userID), nil)
	nsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", options.Token))
	nsReq.Header.Add("Content-Type", "application/json")
	if authReqErr != nil {
		return nil, authReqErr

	}
	nsResp, nsRespErr := client.Do(nsReq)
	if nsRespErr != nil {
		return nil, authReqErr

	}
	defer nsResp.Body.Close()
	if nsResp.StatusCode != 200 {
		var data map[string]interface{}
		json.NewDecoder(nsResp.Body).Decode(&data)
		return nil, fmt.Errorf("Failed to get namespace: %s", data["message"].(string))
	}

	var nsResult map[string]terraUser.User
	json.NewDecoder(nsResp.Body).Decode(&nsResult)
	nsData := nsResult["user"]
	return &nsData, nil
}

// ShowUser displays the user info
func ShowUser(options OptionsDef, userID string) error {
	data, err := GetUser(options, userID)
	if err != nil {
		return err
	}
	data.Password = "*****"
	yamlData, _ := yaml.Marshal(data)
	fmt.Printf("%s\n", yamlData)

	return nil
}

// SetUserPassword modifies user password
func SetUserPassword(options OptionsDef, userID string, password string) error {
	passwordInfo := make(map[string]string)
	passwordInfo["password"] = password
	data, _ := json.Marshal(passwordInfo)
	client := http.Client{}
	nsReq, authReqErr := http.NewRequest("PUT", fmt.Sprintf("%s/auth/user/%s/password", options.URL, userID), bytes.NewBuffer(data))
	nsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", options.Token))
	nsReq.Header.Add("Content-Type", "application/json")
	if authReqErr != nil {
		return authReqErr

	}
	nsResp, nsRespErr := client.Do(nsReq)
	if nsRespErr != nil {
		return authReqErr

	}
	defer nsResp.Body.Close()
	if nsResp.StatusCode != 200 {
		var data map[string]interface{}
		json.NewDecoder(nsResp.Body).Decode(&data)
		return fmt.Errorf("Failed to update user password: %s", data["message"].(string))
	}
	return nil
}

// GetRecipes returns recipes for namespace or public recipes if id is empty
func GetRecipes(options OptionsDef, id string) ([]terraModel.Recipe, error) {
	client := http.Client{}
	nsReq, authReqErr := http.NewRequest("GET", fmt.Sprintf("%s/deploy/recipes", options.URL), nil)
	if id != "" {
		nsReq, authReqErr = http.NewRequest("GET", fmt.Sprintf("%s/deploy/ns/%s/endpoint", options.URL, id), nil)
	}
	nsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", options.Token))
	nsReq.Header.Add("Content-Type", "application/json")

	if authReqErr != nil {
		return nil, authReqErr

	}
	nsResp, nsRespErr := client.Do(nsReq)
	if nsRespErr != nil {
		return nil, authReqErr

	}
	defer nsResp.Body.Close()
	if nsResp.StatusCode != 200 {
		var data map[string]interface{}
		json.NewDecoder(nsResp.Body).Decode(&data)
		return nil, fmt.Errorf("Failed to get endpoints: %s", data["message"].(string))
	}

	var nsResult map[string][]terraModel.Recipe
	json.NewDecoder(nsResp.Body).Decode(&nsResult)
	data := nsResult["recipes"]
	return data, nil
}

// GetRecipe returns selected ns
func GetRecipe(options OptionsDef, nsID string, id string) (*terraModel.Recipe, error) {
	client := http.Client{}
	nsReq, authReqErr := http.NewRequest("GET", fmt.Sprintf("%s/deploy/ns/%s/recipe/%s", options.URL, nsID, id), nil)
	nsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", options.Token))
	nsReq.Header.Add("Content-Type", "application/json")
	if authReqErr != nil {
		return nil, authReqErr

	}
	nsResp, nsRespErr := client.Do(nsReq)
	if nsRespErr != nil {
		return nil, authReqErr

	}
	defer nsResp.Body.Close()
	if nsResp.StatusCode != 200 {
		var data map[string]interface{}
		json.NewDecoder(nsResp.Body).Decode(&data)
		return nil, fmt.Errorf("Failed to get namespace: %s", data["message"].(string))
	}

	var nsResult map[string]terraModel.Recipe
	json.NewDecoder(nsResp.Body).Decode(&nsResult)
	nsData := nsResult["recipe"]
	return &nsData, nil
}

// ListRecipes list the recipes
func ListRecipes(options OptionsDef, nsID string) error {
	data, err := GetRecipes(options, nsID)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, '\t', tabwriter.AlignRight|tabwriter.Debug)
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", "ID", "Name", "Description", "Public", "Namespace")
	for _, ep := range data {
		fmt.Fprintf(w, "%s\t%s\t%s\t%t\t%s\n", ep.ID.Hex(), ep.Name, ep.Description, ep.Public, ep.Namespace)
	}
	w.Flush()
	return nil
}

// ShowRecipe displays the recipe
func ShowRecipe(options OptionsDef, nsID string, id string) error {

	data, err := GetRecipe(options, nsID, id)

	if err != nil {
		return err
	}
	fmt.Printf("id: %s\n", data.ID.Hex())
	yamlData, _ := yaml.Marshal(data)
	fmt.Printf("%s\n", yamlData)
	return nil
}

// GetTemplates returns templates for namespace or public templates if id is empty
func GetTemplates(options OptionsDef, id string) ([]terraModel.Template, error) {
	client := http.Client{}
	nsReq, authReqErr := http.NewRequest("GET", fmt.Sprintf("%s/deploy/templates", options.URL), nil)
	if id != "" {
		nsReq, authReqErr = http.NewRequest("GET", fmt.Sprintf("%s/deploy/ns/%s/template", options.URL, id), nil)
	}
	nsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", options.Token))
	nsReq.Header.Add("Content-Type", "application/json")

	if authReqErr != nil {
		return nil, authReqErr

	}
	nsResp, nsRespErr := client.Do(nsReq)
	if nsRespErr != nil {
		return nil, authReqErr

	}
	defer nsResp.Body.Close()
	if nsResp.StatusCode != 200 {
		var data map[string]interface{}
		json.NewDecoder(nsResp.Body).Decode(&data)
		return nil, fmt.Errorf("Failed to get templates: %s", data["message"].(string))
	}

	var nsResult map[string][]terraModel.Template
	json.NewDecoder(nsResp.Body).Decode(&nsResult)
	data := nsResult["templates"]
	return data, nil
}

// GetTemplate returns selected template
func GetTemplate(options OptionsDef, nsID string, id string) (*terraModel.Template, error) {
	client := http.Client{}
	nsReq, authReqErr := http.NewRequest("GET", fmt.Sprintf("%s/deploy/ns/%s/template/%s", options.URL, nsID, id), nil)
	nsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", options.Token))
	nsReq.Header.Add("Content-Type", "application/json")
	if authReqErr != nil {
		return nil, authReqErr

	}
	nsResp, nsRespErr := client.Do(nsReq)
	if nsRespErr != nil {
		return nil, authReqErr

	}
	defer nsResp.Body.Close()
	if nsResp.StatusCode != 200 {
		var data map[string]interface{}
		json.NewDecoder(nsResp.Body).Decode(&data)
		return nil, fmt.Errorf("Failed to get namespace: %s", data["message"].(string))
	}

	var nsResult map[string]terraModel.Template
	json.NewDecoder(nsResp.Body).Decode(&nsResult)
	nsData := nsResult["template"]
	return &nsData, nil
}

// ListTemplates list the templates
func ListTemplates(options OptionsDef, nsID string) error {
	data, err := GetTemplates(options, nsID)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, '\t', tabwriter.AlignRight|tabwriter.Debug)
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", "ID", "Name", "Description", "Public", "Namespace")
	for _, ep := range data {
		fmt.Fprintf(w, "%s\t%s\t%s\t%t\t%s\n", ep.ID.Hex(), ep.Name, ep.Description, ep.Public, ep.Namespace)
	}
	w.Flush()
	return nil
}

// ShowTemplate displays the template
func ShowTemplate(options OptionsDef, nsID string, id string) error {

	data, err := GetTemplate(options, nsID, id)

	if err != nil {
		return err
	}
	fmt.Printf("id: %s\n", data.ID.Hex())
	yamlData, _ := yaml.Marshal(data)
	fmt.Printf("%s\n", yamlData)
	return nil
}

// GetApps returns apps for namespace or public apps if id is empty
func GetApps(options OptionsDef, id string) ([]terraModel.Application, error) {
	client := http.Client{}
	nsReq, authReqErr := http.NewRequest("GET", fmt.Sprintf("%s/deploy/apps", options.URL), nil)
	if id != "" {
		nsReq, authReqErr = http.NewRequest("GET", fmt.Sprintf("%s/deploy/ns/%s/app", options.URL, id), nil)
	}
	nsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", options.Token))
	nsReq.Header.Add("Content-Type", "application/json")

	if authReqErr != nil {
		return nil, authReqErr

	}
	nsResp, nsRespErr := client.Do(nsReq)
	if nsRespErr != nil {
		return nil, authReqErr

	}
	defer nsResp.Body.Close()
	if nsResp.StatusCode != 200 {
		var data map[string]interface{}
		json.NewDecoder(nsResp.Body).Decode(&data)
		return nil, fmt.Errorf("Failed to get applications: %s", data["message"].(string))
	}

	var nsResult map[string][]terraModel.Application
	json.NewDecoder(nsResp.Body).Decode(&nsResult)
	data := nsResult["apps"]
	return data, nil
}

// GetApp returns selected template
func GetApp(options OptionsDef, nsID string, id string) (*terraModel.Application, error) {
	client := http.Client{}
	nsReq, authReqErr := http.NewRequest("GET", fmt.Sprintf("%s/deploy/ns/%s/app/%s", options.URL, nsID, id), nil)
	nsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", options.Token))
	nsReq.Header.Add("Content-Type", "application/json")
	if authReqErr != nil {
		return nil, authReqErr

	}
	nsResp, nsRespErr := client.Do(nsReq)
	if nsRespErr != nil {
		return nil, authReqErr

	}
	defer nsResp.Body.Close()
	if nsResp.StatusCode != 200 {
		var data map[string]interface{}
		json.NewDecoder(nsResp.Body).Decode(&data)
		return nil, fmt.Errorf("Failed to get application: %s", data["message"].(string))
	}

	var nsResult map[string]terraModel.Application
	json.NewDecoder(nsResp.Body).Decode(&nsResult)
	nsData := nsResult["app"]
	return &nsData, nil
}

// ListApps list the applications
func ListApps(options OptionsDef, nsID string) error {
	data, err := GetApps(options, nsID)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, '\t', tabwriter.AlignRight|tabwriter.Debug)
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", "ID", "Name", "Description", "Public", "Namespace")
	for _, ep := range data {
		fmt.Fprintf(w, "%s\t%s\t%s\t%t\t%s\n", ep.ID.Hex(), ep.Name, ep.Description, ep.Public, ep.Namespace)
	}
	w.Flush()
	return nil
}

// ShowApp displays the application
func ShowApp(options OptionsDef, nsID string, id string) error {

	data, err := GetApp(options, nsID, id)

	if err != nil {
		return err
	}
	fmt.Printf("id: %s\n", data.ID.Hex())
	yamlData, _ := yaml.Marshal(data)
	fmt.Printf("%s\n", yamlData)
	return nil
}

// *******************
func appInputs(options OptionsDef, nsID string, appID string) (map[string]interface{}, error) {
	client := http.Client{}

	nsReq, runReqErr := http.NewRequest("GET", fmt.Sprintf("%s/deploy/ns/%s/app/%s/inputs", options.URL, nsID, appID), nil)
	nsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", options.Token))
	nsReq.Header.Add("Content-Type", "application/json")

	if runReqErr != nil {
		return nil, runReqErr

	}
	nsResp, nsRespErr := client.Do(nsReq)
	if nsRespErr != nil {
		return nil, nsRespErr

	}
	defer nsResp.Body.Close()
	if nsResp.StatusCode != 200 {
		var data map[string]interface{}
		json.NewDecoder(nsResp.Body).Decode(&data)
		return nil, fmt.Errorf("Failed to get application inputs: %s", data["message"].(string))
	}

	var res map[string]map[string]interface{}
	json.NewDecoder(nsResp.Body).Decode(&res)
	inputs := res["app"]

	//respData, _ := ioutil.ReadAll(nsResp.Body)
	//fmt.Printf("app inputs: %s", respData)
	return inputs, nil

}

func endpointDefaultInputs(options OptionsDef, nsID string, endpointID string) (map[string][]string, error) {
	client := http.Client{}
	nsReq, runReqErr := http.NewRequest("GET", fmt.Sprintf("%s/deploy/ns/%s/endpoint/%s/defaults", options.URL, nsID, endpointID), nil)
	nsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", options.Token))
	nsReq.Header.Add("Content-Type", "application/json")

	if runReqErr != nil {
		return nil, runReqErr

	}
	nsResp, nsRespErr := client.Do(nsReq)
	if nsRespErr != nil {
		return nil, nsRespErr

	}
	defer nsResp.Body.Close()
	if nsResp.StatusCode != 200 {
		var data map[string]interface{}
		json.NewDecoder(nsResp.Body).Decode(&data)
		return nil, fmt.Errorf("Failed to get endpoint defaults: %s", data["message"].(string))
	}
	var endpointDefaults map[string]map[string][]string
	json.NewDecoder(nsResp.Body).Decode(&endpointDefaults)
	return endpointDefaults["defaults"], nil
}

func promptUser(label string) string {
	fmt.Printf("%s: ", label)
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	text = strings.Replace(text, "\n", "", -1)
	return text
}

// RunInputs contains run input parameters
type RunInputs struct {
	Params map[string]string `yaml:"params"`
}

func hasSecret(options OptionsDef, nsID string, endpointID string) bool {
	client := http.Client{}
	nsReq, runReqErr := http.NewRequest("GET", fmt.Sprintf("%s/deploy/ns/%s/endpoint/%s/secret", options.URL, nsID, endpointID), nil)
	nsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", options.Token))
	nsReq.Header.Add("Content-Type", "application/json")

	if runReqErr != nil {

		return false

	}
	nsResp, nsRespErr := client.Do(nsReq)
	if nsRespErr != nil {
		return false

	}
	defer nsResp.Body.Close()
	if nsResp.StatusCode != 200 {
		return false
	}
	return true
}

func runRun(options OptionsDef, run terraModel.Run) (string, error) {
	client := http.Client{}
	data, _ := json.Marshal(run)
	fmt.Printf("Run %+v\n", run)
	fmt.Printf("%s/deploy/ns/%s/run/%s\n", options.URL, run.Endpoint, run.AppID)
	nsReq, runReqErr := http.NewRequest("POST", fmt.Sprintf("%s/deploy/ns/%s/run/%s", options.URL, run.Namespace, run.AppID), bytes.NewBuffer(data))
	nsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", options.Token))
	nsReq.Header.Add("Content-Type", "application/json")

	if runReqErr != nil {
		return "", runReqErr

	}
	nsResp, nsRespErr := client.Do(nsReq)
	if nsRespErr != nil {
		return "", nsRespErr

	}
	defer nsResp.Body.Close()
	if nsResp.StatusCode != 201 {
		var data map[string]interface{}
		json.NewDecoder(nsResp.Body).Decode(&data)
		fmt.Printf("### %d, %+v", nsResp.StatusCode, data)
		return "", fmt.Errorf("Failed to run application: %s", data["message"].(string))
	}
	var runRespData map[string]string
	json.NewDecoder(nsResp.Body).Decode(&runRespData)

	return runRespData["run"], nil
}

// StartRun exec a run
func StartRun(options OptionsDef, name string, nsID string, endpointID string, appID string, params string, template bool) (string, error) {
	paramData := make(map[string]string)

	hasSecret := hasSecret(options, nsID, endpointID)
	if !hasSecret {
		return "", fmt.Errorf("no known secret for this endpoint, please create one first")
	}

	if params == "" {

		endpointInfo, _ := GetEndpoint(options, nsID, endpointID)

		inputs, inputsErr := appInputs(options, nsID, appID)
		if inputsErr != nil {
			return "", inputsErr
		}

		endpointDefaultInputParams, endpointDefaultInputError := endpointDefaultInputs(options, nsID, endpointID)

		var defaultInputs map[string]interface{}
		defaultInputs = inputs["defaults"].(map[string]interface{})
		templateInputs := inputs["template"].(map[string]interface{})
		fmt.Println("Template parameters:")
		for param, paramLabel := range templateInputs {
			paramData[param] = ""
			// fmt.Printf("%s ?\n", paramLabel.(string))
			defaults, ok := defaultInputs[param]
			if ok {
				if len(defaults.([]string)) == 1 {
					fmt.Printf("Default: %s\n", defaults.([]string)[0])
					paramData[param] = defaults.([]string)[0]
				} else {
					fmt.Printf("Choices: %s\n", strings.Join(defaults.([]string), ","))
				}
			}
			if endpointDefaultInputError == nil {
				// fmt.Printf("has default? %s, %+v", param, endpointDefaultInputParams)
				epDefaults, ok := endpointDefaultInputParams[param]
				if ok {
					if len(epDefaults) == 1 {
						fmt.Printf("Default: %s\n", epDefaults[0])
						paramData[param] = epDefaults[0]
					} else {
						fmt.Printf("Choices: %s\n", strings.Join(epDefaults, ","))
					}
				}
			}
			if paramData[param] == "" {
				paramData[param] = promptUser(paramLabel.(string))
			}
		}
		fmt.Println("Recipe parameters:")
		for param, paramLabel := range inputs["recipes"].(map[string]interface{}) {
			paramData[param] = ""
			// fmt.Printf("%s ?\n", paramLabel.(string))
			defaults, ok := defaultInputs[param]
			if ok {
				if len(defaults.([]string)) == 1 {
					fmt.Printf("Default: %s\n", defaults.([]string)[0])
					paramData[param] = defaults.([]string)[0]
				} else {
					fmt.Printf("Choices: %s\n", strings.Join(defaults.([]string), ","))
				}
			}
			if endpointDefaultInputError == nil {
				// fmt.Printf("has default? %s, %+v", param, endpointDefaultInputParams)
				epDefaults, ok := endpointDefaultInputParams[param]
				if ok {
					if len(epDefaults) == 1 {
						fmt.Printf("Default: %s\n", epDefaults[0])
						paramData[param] = epDefaults[0]
					} else {
						fmt.Printf("Choices: %s\n", strings.Join(epDefaults, ","))
					}
				}
			}
			if paramData[param] == "" {
				paramData[param] = promptUser(paramLabel.(string))
			}
		}
		endpoints := inputs["endpoints"].(map[string]interface{})
		endpointInputs := endpoints[endpointInfo.Name].(map[string]interface{})

		fmt.Println("Endpoint parameters:")
		for param, paramLabel := range endpointInputs {
			paramData[param] = ""
			// fmt.Printf("%s ?\n", paramLabel)
			// fmt.Printf("defaults=%+v\n", endpointDefaultInputParams)
			defaults, ok := defaultInputs[param]
			if ok {
				if len(defaults.([]string)) == 1 {
					fmt.Printf("Default: %s\n", defaults.([]string)[0])
					paramData[param] = defaults.([]string)[0]
				} else {
					fmt.Printf("Choices: %s\n", strings.Join(defaults.([]string), ","))
				}
			}
			if endpointDefaultInputError == nil {
				// fmt.Printf("has default? %s, %+v", param, endpointDefaultInputParams)
				epDefaults, ok := endpointDefaultInputParams[param]
				if ok {
					if len(epDefaults) == 1 {
						fmt.Printf("Default: %s\n", epDefaults[0])
						paramData[param] = epDefaults[0]
					} else {
						fmt.Printf("Choices: %s\n", strings.Join(epDefaults, ","))
					}
				}
			}
			if paramData[param] == "" {
				paramData[param] = promptUser(paramLabel.(string))
			}
		}

	} else {
		var runconfig RunInputs
		cfg, err := ioutil.ReadFile(params)
		if err != nil {
			return "", err
		}
		yaml.Unmarshal([]byte(cfg), &runconfig)
		paramData = runconfig.Params

	}

	if template {
		runconfig := RunInputs{}
		runconfig.Params = paramData
		yamlData, _ := yaml.Marshal(runconfig)
		fmt.Printf("\nYaml parameters template:\n%s\n", yamlData)
		return "", nil
	}

	sensitive := make(map[string]string)
	runInputData := terraModel.Run{Name: name, Namespace: nsID, Inputs: paramData, Endpoint: endpointID, AppID: appID, SensitiveInputs: sensitive}
	runID, runError := runRun(options, runInputData)

	return runID, runError
}

// GetRuns returns user runs
func GetRuns(options OptionsDef, id string) ([]terraModel.Run, error) {
	client := http.Client{}
	nsReq, authReqErr := http.NewRequest("GET", fmt.Sprintf("%s/deploy/run", options.URL), nil)
	if id != "" {
		nsReq, authReqErr = http.NewRequest("GET", fmt.Sprintf("%s/deploy/ns/%s/run", options.URL, id), nil)
	}
	nsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", options.Token))
	nsReq.Header.Add("Content-Type", "application/json")

	if authReqErr != nil {
		return nil, authReqErr

	}
	nsResp, nsRespErr := client.Do(nsReq)
	if nsRespErr != nil {
		return nil, nsRespErr

	}
	defer nsResp.Body.Close()
	if nsResp.StatusCode != 200 {
		var data map[string]interface{}
		json.NewDecoder(nsResp.Body).Decode(&data)
		return nil, fmt.Errorf("Failed to get applications: %s", data["message"].(string))
	}

	var nsResult map[string][]terraModel.Run
	json.NewDecoder(nsResp.Body).Decode(&nsResult)
	data := nsResult["runs"]
	return data, nil
}

// GetRun returns selected run
func GetRun(options OptionsDef, nsID, id string) (*terraModel.Run, error) {
	client := http.Client{}
	nsReq, authReqErr := http.NewRequest("GET", fmt.Sprintf("%s/deploy/ns/%s/run/%s", options.URL, nsID, id), nil)
	nsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", options.Token))
	nsReq.Header.Add("Content-Type", "application/json")
	if authReqErr != nil {
		return nil, authReqErr

	}
	nsResp, nsRespErr := client.Do(nsReq)
	if nsRespErr != nil {
		return nil, authReqErr

	}
	defer nsResp.Body.Close()
	if nsResp.StatusCode != 200 {
		var data map[string]interface{}
		json.NewDecoder(nsResp.Body).Decode(&data)
		return nil, fmt.Errorf("Failed to get application: %s", data["message"].(string))
	}

	var nsData terraModel.Run
	json.NewDecoder(nsResp.Body).Decode(&nsData)
	return &nsData, nil
}

// DeleteRun ask for run termination
func DeleteRun(options OptionsDef, nsID string, id string) error {
	client := http.Client{}

	nsReq, authReqErr := http.NewRequest("DELETE", fmt.Sprintf("%s/deploy/ns/%s/run/%s", options.URL, nsID, id), nil)

	nsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", options.Token))
	nsReq.Header.Add("Content-Type", "application/json")

	if authReqErr != nil {
		return authReqErr

	}
	nsResp, nsRespErr := client.Do(nsReq)
	if nsRespErr != nil {
		return authReqErr

	}
	defer nsResp.Body.Close()
	if nsResp.StatusCode != 200 {
		var data map[string]interface{}
		json.NewDecoder(nsResp.Body).Decode(&data)
		return fmt.Errorf("Failed to get applications: %s", data["message"].(string))
	}

	return nil
}

// ListRuns list the user runs
func ListRuns(options OptionsDef, nsID string) error {
	data, err := GetRuns(options, nsID)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, '\t', tabwriter.AlignRight|tabwriter.Debug)
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", "ID", "Name", "Status", "Start", "End", "Namespace")
	for _, ep := range data {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", ep.ID.Hex(), ep.Name, ep.Status, time.Unix(ep.Start, 0), time.Unix(ep.End, 0), ep.Namespace)
	}
	w.Flush()
	return nil
}

// GetRunStore returns run deployment data in store
func GetRunStore(options OptionsDef, id string) (*map[string]interface{}, error) {
	client := http.Client{}
	nsReq, authReqErr := http.NewRequest("GET", fmt.Sprintf("%s/store/%s", options.URL, id), nil)
	nsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", options.Token))
	nsReq.Header.Add("Content-Type", "application/json")
	if authReqErr != nil {
		return nil, authReqErr

	}
	nsResp, nsRespErr := client.Do(nsReq)
	if nsRespErr != nil {
		return nil, authReqErr

	}
	defer nsResp.Body.Close()
	if nsResp.StatusCode != 200 {
		var data map[string]interface{}
		json.NewDecoder(nsResp.Body).Decode(&data)
		return nil, fmt.Errorf("Failed to get application: %s", data["message"].(string))
	}

	var nsData map[string]interface{}
	json.NewDecoder(nsResp.Body).Decode(&nsData)
	return &nsData, nil
}

// ShowRun displays the run info
func ShowRun(options OptionsDef, nsID string, id string, store bool) error {

	data, err := GetRun(options, nsID, id)

	if err != nil {
		return err
	}
	fmt.Printf("id: %s\n", data.ID.Hex())
	yamlData, _ := yaml.Marshal(data)
	fmt.Printf("%s\n", yamlData)

	fmt.Println("Store data")
	if data.Deployment == "" {
		fmt.Println("\tno data")
	} else {
		storeData, errData := GetRunStore(options, data.Deployment)
		if errData != nil {
			return errData
		}
		yamlData, _ = yaml.Marshal(storeData)
		fmt.Printf("%s\n", yamlData)
	}
	return nil
}
