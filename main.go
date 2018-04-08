package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"

	"gopkg.in/ldap.v2"
)

const letterBytes = "0123456789ABCDEF"

type Connection struct {
	url      string
	login    string
	password string
}

type Users struct {
	UsersList []User `json:"user"`
}

type User struct {
	Id       int    `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Href     string `json:"href"`
	Mail     string `json:"email"`
}

type Groups struct {
	GroupList []Group `json:"group"`
}

type Group struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	Href  string `json:"href"`
	Users Users  `json:"users"`
}

func getLDAPUsers(groupName, baseDN string, link *ldap.Conn) []User {

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
		userList = append(userList, getLDAPUserAttributes(entry.DN, baseDN, link))
	}

	return userList
}

func getLDAPUserAttributes(userDN, baseDN string, link *ldap.Conn) User {
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

func getGroupDN(groupName, baseDN string, link *ldap.Conn) ([]string, []string) {

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

	var dnList []string
	var cnList []string

	for _, group := range sr.Entries {
		dnList = append(dnList, group.DN)
		cnList = append(cnList, group.GetAttributeValues("cn")[0])
	}

	return dnList, cnList
}

func getTCGroups(conn Connection, client http.Client) Groups {

	url := conn.url + "/app/rest/userGroups"
	searcherReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err)
	}
	searcherReq.Header.Add("Content-type", "application/json")
	searcherReq.Header.Add("Accept", "application/json")
	searcherReq.SetBasicAuth(conn.login, conn.password)

	resp, err := client.Do(searcherReq)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}

	var groups Groups
	err = json.Unmarshal(body, &groups)
	if err != nil {
		log.Println(err)
	}

	return groups
}

func getTCUsers(conn Connection, client http.Client) Users {

	url := conn.url + "/app/rest/users"
	searcherReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err)
	}
	searcherReq.Header.Add("Content-type", "application/json")
	searcherReq.Header.Add("Accept", "application/json")
	searcherReq.SetBasicAuth(conn.login, conn.password)

	resp, err := client.Do(searcherReq)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}

	var users Users
	err = json.Unmarshal(body, &users)
	if err != nil {
		log.Println(err)
	}
	return users
}

func (group Group) getUsersFromGroup(conn Connection, client http.Client) Users {

	url := conn.url + group.Href
	searcherReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err)
	}
	searcherReq.Header.Add("Content-type", "application/json")
	searcherReq.Header.Add("Accept", "application/json")
	searcherReq.SetBasicAuth(conn.login, conn.password)

	resp, err := client.Do(searcherReq)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}

	var users Group
	err = json.Unmarshal(body, &users)
	if err != nil {
		log.Println(err)
	}

	return users.Users
}

func createGroup(groupName string, conn Connection, client http.Client) {
	fmt.Println("Creating group", groupName)
	var group Group
	url := conn.url + "/app/rest/userGroups"

	group.Key = RandStringBytes(16)
	group.Name = groupName

	data, err := json.Marshal(group)

	if err != nil {
		panic(err)
	}

	searcherReq, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	searcherReq.Header.Add("Content-type", "application/json")
	searcherReq.Header.Add("Accept", "application/json")
	searcherReq.SetBasicAuth(conn.login, conn.password)

	resp, err := client.Do(searcherReq)
	if err != nil {
		// panic(err)
		fmt.Println("response Status:", resp.Status)
		fmt.Println("response Headers:", resp.Header)
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println("response Body:", string(body))
	}
	defer resp.Body.Close()
}

func (user User) getUserGroups(conn Connection, client http.Client) Groups {

	url := conn.url + "/app/rest/users/" + user.Username + "/groups"
	searcherReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err)
	}
	searcherReq.Header.Add("Content-type", "application/json")
	searcherReq.Header.Add("Accept", "application/json")
	searcherReq.SetBasicAuth(conn.login, conn.password)

	resp, err := client.Do(searcherReq)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}

	var userGroups Groups
	err = json.Unmarshal(body, &userGroups)
	if err != nil {
		log.Println(err)
	}

	return userGroups
}

func (user User) addUserToGroup(conn Connection, client http.Client) {
	//def add_user_to_group(self, user, group_name):
	//print("Adding user {} to group {}".format(user, group_name))
	//url = self.rest_url + 'users/' + user + '/groups'
	//   user_groups = TeamCityClient.get_user_groups(self, user)
	//href = [group['href'] for group in self.tc_groups if group['name'] == group_name][0]
	//key = [group['key'] for group in self.tc_groups if group['name'] == group_name][0]
	//new_group = {u'href': href,
	//	u'name': group_name,
	//	u'key': key}
	//user_groups['group'].append(new_group)
	//data = json.dumps(user_groups)
	//resp = self.session.put(url, data=data, verify=False)
	//if resp.status_code != 200:
	//print("Error: Couldn't add user {} to group {}\n{}".format(user, group_name, resp.content))
}

func (user User) removeUserFromGroup(conn Connection, client http.Client) {
	//print("Removing user {} from group {}".format(user, group_name))
	//url = self.rest_url + 'users/' + user + '/groups'
	//   user_groups = TeamCityClient.get_user_groups(self, user)
	//for group in user_groups['group']:
	//if group['name'] == group_name:
	//user_groups['group'].remove(group)
	//data = json.dumps(user_groups)
	//resp = self.session.put(url, data=data, verify=False)
	//if resp.status_code != 200:
	//print("Error: Couldn't remove user {} from group {}\n{}".format(user, group_name, resp.content))
}

func (user User) createUser(conn Connection, client http.Client, wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Println("Creating user", user.Username)
	url := conn.url + "/app/rest/users"
	data, err := json.Marshal(user)
	if err != nil {
		panic(err)
	}

	searcherReq, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		log.Println(err)
	}
	searcherReq.Header.Add("Content-type", "application/json")
	searcherReq.Header.Add("Accept", "application/json")
	searcherReq.SetBasicAuth(conn.login, conn.password)

	resp, err := client.Do(searcherReq)
	if err != nil {
		// panic(err)
		fmt.Println("response Status:", resp.Status)
		fmt.Println("response Headers:", resp.Header)
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println("response Body:", string(body))
	}
	defer resp.Body.Close()
}

func userInTC(ldapUser User, tcUsers Users) bool {
	for _, tcUser := range tcUsers.UsersList {
		if strings.ToLower(tcUser.Username) == strings.ToLower(ldapUser.Username) {
			return true
		}
	}
	return false
}

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func main() {
	username := flag.String("username", "username@domain.com", "Domain login for auth")
	password := flag.String("password", "topSecret", "Password for auth")
	server := flag.String("server", "domain.com", "Address of LDAP server")
	// tcServer := flag.String("tcServer", "https://teamcity.domain.com", "Address of LDAP server")
	port := flag.String("port", "389", "Port of LDAP server")
	// tcUser := flag.String("tcUser", "", "User for TC with admin permissions")
	// tcPassword := flag.String("tcPassword", "", "User for TC with admin permissions")
	flag.Parse()

	// No TLS, not recommended
	l, err := ldap.Dial("tcp", fmt.Sprintf("%s:%s", *server, *port))
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	err = l.Bind(*username, *password)
	if err != nil {
		log.Fatal(err)
	}

	// client := &http.Client{}
	// connection := Connection{*tcServer, *tcUser, *tcPassword}

	// использовать функцию только если есть флаг WILDCARD
	_, groupCNs := getGroupDN("R.MPX.QA*Zabbix*", "dc=ptsecurity,dc=ru", l) // добавить выбор группы, base брать из лдап конекшн
	for _, group := range groupCNs {
		fmt.Println(group)
	}
	// ldapUsers := getLDAPUsers(groupDN, "dc=ptsecurity,dc=ru", l) // base брать из лдап конекшн
	// tcUsers := getTCUsers(connection, *client)

	// userList := getTCUsers(connection, *client)
	// fish := userList.UsersList[0]
	// fish.createUser(connection, *client, wg)

	// wg := &sync.WaitGroup{}

	// for _, ldapUser := range ldapUsers {
	// 	if !userInTC(ldapUser, tcUsers) {
	// 		wg.Add(1)
	// 		go ldapUser.createUser(connection, *client, wg)
	// 	}
	// }
	// wg.Wait()

	//fGroup := fish.getUserGroups(connection, *client)
	//myh := fGroup.GroupList[1]
	//fmt.Println(myh)
	//myh.getUsersFromGroup(connection, *client)
	fmt.Println("Done!")
}
