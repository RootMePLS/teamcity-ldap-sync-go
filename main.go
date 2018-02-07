package main

import (
	"fmt"
	"gopkg.in/ldap.v2"
	"log"
	"flag"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"strings"
)

type Users struct {
	UsersList []User `json:"user"`
}

type User struct {
	Id       int    `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Href     string `json:"href"`
	Mail     string
}

type Groups struct {
	GroupList []Group `json:"group"`
}

type Group struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Href string `json:"href"`
}

func getUsersFromAD(groupName, baseDN string, link ldap.Conn) []User {

	var userList []User

	filter := fmt.Sprintf("(&(objectClass=user)(objectCategory=Person)(memberOf:1.2.840.113556.1.4.1941:=%s))", groupName)
	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		filter,
		[]string{},
		nil,
	)

	sr, err := link.Search(searchRequest)
	if err != nil {
		log.Fatal(err)
	}

	for _, entry := range sr.Entries {
		userList = append(userList, getUserAttributes(entry.DN, baseDN, link))
	}

	return userList
}

func getUserAttributes(userDN, baseDN string, link ldap.Conn) User {
	var user User

	filter := fmt.Sprintf("(distinguishedName=%s)", userDN)
	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		filter,
		[]string{"sn", "givenName", "mail", "sAMAccountName"},
		nil,
	)

	sr, err := link.Search(searchRequest)
	if err != nil {
		log.Fatal(err)
	}

	fullName := sr.Entries[0].GetAttributeValue("givenName") + " " + sr.Entries[0].GetAttributeValue("sn")

	user.Name = fullName
	user.Username = sr.Entries[0].GetAttributeValue("sAMAccountName")
	user.Mail = sr.Entries[0].GetAttributeValue("mail")

	return user
}

func getGroupDN(groupName, baseDN string, link ldap.Conn) string {

	filter := fmt.Sprintf("(&(objectClass=group)(cn=%s))", groupName)
	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		filter,
		[]string{},
		nil,
	)

	sr, err := link.Search(searchRequest)
	if err != nil {
		log.Fatal(err)
	}

	return sr.Entries[0].DN
}

func getTCGroups(url, login, password string, client http.Client) Groups {

	url = "/app/rest/userGroups"
	searcherReq, err := http.NewRequest("GET", url, nil)
	searcherReq.Header.Add("Content-type", "application/json")
	searcherReq.Header.Add("Accept", "application/json")
	searcherReq.SetBasicAuth(login, password)

	resp, err := client.Do(searcherReq)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	var groups Groups
	json.Unmarshal(body, &groups)

	return groups
}

func getTCUsers(url, login, password string, client http.Client) Users {

	url = url + "/app/rest/users"
	searcherReq, err := http.NewRequest("GET", url, nil)
	searcherReq.Header.Add("Content-type", "application/json")
	searcherReq.Header.Add("Accept", "application/json")
	searcherReq.SetBasicAuth(login, password)

	resp, err := client.Do(searcherReq)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	var users Users
	json.Unmarshal(body, &users)

	return users
}

func (u User) getTCUserGroups(url, username, login, password string, client http.Client) Groups {
	url = url + "/app/rest/users/" + username + "/groups"
	searcherReq, err := http.NewRequest("GET", url, nil)
	searcherReq.Header.Add("Content-type", "application/json")
	searcherReq.Header.Add("Accept", "application/json")
	searcherReq.SetBasicAuth(login, password)

	resp, err := client.Do(searcherReq)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	var userGroups Groups
	json.Unmarshal(body, &userGroups)

	return userGroups
}

func main() {

	username := flag.String("username", "username@domain.com", "Domain login for auth")
	password := flag.String("password", "topSecret", "Password for auth")
	server := flag.String("server", "domain.com", "Address of LDAP server")
	teamcity := flag.String("teamcity", "https://teamcity.domain.com", "Address of LDAP server")
	port := flag.String("port", "389", "Port of LDAP server")
	flag.Parse()
	tcUsername := strings.Split(*username, "@")[0]

	// No TLS, not recommended
	l, err := ldap.Dial("tcp", fmt.Sprintf("%s:%s", *server, *port))
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	err = l.Bind(*username, *password)
	if err != nil {
		log.Println(err)
	}

	client := &http.Client{}
	userList := getTCUsers(*teamcity, tcUsername, *password, *client)
	fmt.Println(userList.UsersList[255])
	fish := userList.UsersList[255]
	fGroups := fish.getTCUserGroups(*teamcity, fish.Username, tcUsername, *password, *client)
	fmt.Println(fGroups)
}
