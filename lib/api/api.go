package goterraapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

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
		return nil, fmt.Errorf("Failed to get endpoints: %s", data["message"].(string))
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
func GetRecipe(options OptionsDef, nsID, id string) (*terraModel.Recipe, error) {
	client := http.Client{}
	nsReq, authReqErr := http.NewRequest("GET", fmt.Sprintf("%s/deploy/ns/%s/recipe/%s", options.URL, id, id), nil)
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
func GetTemplate(options OptionsDef, nsID, id string) (*terraModel.Template, error) {
	client := http.Client{}
	nsReq, authReqErr := http.NewRequest("GET", fmt.Sprintf("%s/deploy/ns/%s/template/%s", options.URL, id, id), nil)
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

// *********************************

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
func GetApp(options OptionsDef, nsID, id string) (*terraModel.Application, error) {
	client := http.Client{}
	nsReq, authReqErr := http.NewRequest("GET", fmt.Sprintf("%s/deploy/ns/%s/app/%s", options.URL, id, id), nil)
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
