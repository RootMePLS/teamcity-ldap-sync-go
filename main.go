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
	Mail     string
}

type Groups struct {
	GroupList []Group `json:"group"`
}

type Group struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Href string `json:"href"`
	Users  Users  `json:"users"`
}

func getUsersLDAP(groupName, baseDN string, link ldap.Conn) []User {

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
		userList = append(userList, getUserAttributesLDAP(entry.DN, baseDN, link))
	}

	return userList
}

func getUserAttributesLDAP(userDN, baseDN string, link ldap.Conn) User {
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

func getGroupDNLDAP(groupName, baseDN string, link ldap.Conn) string {

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

func getGroupsTC(conn Connection, client http.Client) Groups {

	url := conn.url + "/app/rest/userGroups"
	searcherReq, err := http.NewRequest("GET", url, nil)
	searcherReq.Header.Add("Content-type", "application/json")
	searcherReq.Header.Add("Accept", "application/json")
	searcherReq.SetBasicAuth(conn.login, conn.password)

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

func getUsersTC(conn Connection, client http.Client) Users {

	url := conn.url + "/app/rest/users"
	searcherReq, err := http.NewRequest("GET", url, nil)
	searcherReq.Header.Add("Content-type", "application/json")
	searcherReq.Header.Add("Accept", "application/json")
	searcherReq.SetBasicAuth(conn.login, conn.password)

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

func (user User) getUserGroups(conn Connection, client http.Client) Groups {
	url := conn.url + "/app/rest/users/" + user.Username + "/groups"
	searcherReq, err := http.NewRequest("GET", url, nil)
	searcherReq.Header.Add("Content-type", "application/json")
	searcherReq.Header.Add("Accept", "application/json")
	searcherReq.SetBasicAuth(conn.login, conn.password)

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

func (group Group) getUsersFromGroup(conn Connection, client http.Client) Users {

	url := conn.url + group.Href
	searcherReq, err := http.NewRequest("GET", url, nil)
	searcherReq.Header.Add("Content-type", "application/json")
	searcherReq.Header.Add("Accept", "application/json")
	searcherReq.SetBasicAuth(conn.login, conn.password)

	resp, err := client.Do(searcherReq)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	var users Group
	json.Unmarshal(body, &users)

	return users.Users
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

func createUser(conn Connection, client http.Client) {
	//def create_group(self, group_name):
	//print("Creating group {} in TC".format(group_name))
	//url = self.rest_url + 'userGroups'
	//   key = ''.join(random.choice('0123456789ABCDEF') for i in range(16))
	//data = json.dumps({"name": group_name, "key": key})
	//resp = self.session.post(url, verify=False, data=data)
	//if resp.status_code == 200:
	//self.tc_groups = TeamCityClient.get_tc_groups(self)
	//else:
	//print("Error: Couldn't create group {}\n{}".format(group_name, resp.content))
}

func createGroup(conn Connection, client http.Client) {
	//def create_user(self, user):
	//print("Creating user {}".format(user['username']))
	//url = self.rest_url + 'users'
	//   if not user['email']:
	//user['email'] = ''
	//data = json.dumps({u'username': user['username'], u'name': user['name'], u'email': user['email']})
	//
	//resp = self.session.post(url, verify=False, data=data)
	//if resp.status_code == 200:
	//self.tc_users = TeamCityClient.get_tc_users(self)
	//else:
	//print("Error: Couldn't create user {}\n{}".format(user['username'], resp.content))
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
	connection := Connection{*teamcity, tcUsername, *password}

	userList := getUsersTC(connection, *client)
	fish := userList.UsersList[255]
	fGroup := fish.getUserGroups(connection, *client)
	myh := fGroup.GroupList[1]
	fmt.Println(myh)
	myh.getUsersFromGroup(connection, *client)
}
