package main

import "fmt"
import (
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

func getTCGroups(url, username, password string, client http.Client) Groups {

	url = "/app/rest/userGroups"
	searcherReq, err := http.NewRequest("GET", url, nil)
	searcherReq.Header.Add("Content-type", "application/json")
	searcherReq.Header.Add("Accept", "application/json")
	searcherReq.SetBasicAuth(username, password)

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

func getTCUsers(url, username, password string, client http.Client) Users {

	url = url + "/app/rest/users"
	searcherReq, err := http.NewRequest("GET", url, nil)
	searcherReq.Header.Add("Content-type", "application/json")
	searcherReq.Header.Add("Accept", "application/json")
	searcherReq.SetBasicAuth(username, password)

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

	////group := "R.MPX.ISIM.Repos.All.Reader"
	//group := "CN=R.MPX.ISIM.Repos.All.Reader,OU=DevOpsRoles,OU=Groups,OU=Msk,DC=ptsecurity,DC=ru"
	////group := "DevOps"
	////filter := fmt.Sprintf("(&(objectClass=user)(memberof:1.2.840.113556.1.4.1941:=%s))", "*Zabbix*")
	//filter := fmt.Sprintf("(&(objectClass=user)(objectCategory=Person)(memberOf:1.2.840.113556.1.4.1941:=%s))", group)
	////filter := fmt.Sprintf("(distinguishedName=%s)", "CN=Vasiliy Zvyagintsev,OU=Users,OU=Tmsk,DC=ptsecurity,DC=ru")
	////filter := fmt.Sprintf("(sAMAccountName=%s)", dn)
	////filter := fmt.Sprintf("(&(objectClass=group)(cn=%s))", group)
	//searchRequest := ldap.NewSearchRequest(
	//	"dc=ptsecurity,dc=ru", // The base dn to search
	//	ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
	//	filter,                        // The filter to apply
	//	[]string{"dn", "cn", "member", "sn", "givenName", "mail"}, // A list attributes to retrieve
	//	nil,
	//)
	//
	//sr, err := l.Search(searchRequest)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//for _, entry := range sr.Entries { // [0].Attributes[1].Values {
	//	//fmt.Println(entry.GetAttributeValues("member"))
	//	//fmt.Println(entry.GetAttributeValues("mail"))
	//	//fmt.Println(entry.GetAttributeValues("dn"))
	//	//fmt.Println(entry.GetAttributeValues("cn"))
	//	//fmt.Println(entry.GetAttributeValues("sn"))
	//	fmt.Println(entry.DN)
	//	//fmt.Println(entry.GetAttributeValues("member")[0])
	//	//fmt.Println(entry.Attributes[0])
	//	//fmt.Printf("%s: %v\n", entry.DN, entry.GetAttributeValue("member"))
	//}

	//groups := []string{"R.MPX.ISIM.Repos.All.Reader",}
	//for _, groupName := range groups {
	//	groupDN := getGroupDN(groupName, "dc=ptsecurity,dc=ru", *l)
	//	users := getUsersFromAD(groupDN, "dc=ptsecurity,dc=ru", *l)
	//	fmt.Println(users, "\n" ,len(users))
	//}
	//getTCGroups()
	//
	client := &http.Client{}
	userList := getTCUsers(*teamcity, tcUsername, *password, *client)
	fmt.Println(userList)
}
