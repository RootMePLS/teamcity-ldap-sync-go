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
	"runtime"
	"strings"
	"sync"
	"time"

	"gopkg.in/ldap.v2"
)

type Connection struct {
	url      string
	login    string
	password string
}

type Users struct {
	UsersList []User `json:"user,omitempty"`
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
	Key         string `json:"key"`
	Name        string `json:"name"`
	Href        string `json:"href,omitempty"`
	Description string `json:"description,omitempty"`
	Users       *Users `json:"users,omitempty"`
}

// TODO: есть ощущение, что количество кода можно сильно сократить через одну функцию которая принимает реквест и параметр, а внутри select'ом решает, что-куда

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

func getGroupDN(groupName, baseDN string, link *ldap.Conn) map[string]string {

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

	ldapGroups := make(map[string]string)

	for _, group := range sr.Entries {
		ldapGroups[group.GetAttributeValues("cn")[0]] = group.DN
	}

	return ldapGroups
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

	var raw_groups Groups
	err = json.Unmarshal(body, &raw_groups)
	if err != nil {
		log.Println(err)
	}

	// var groups []string
	// for _, group := range raw_groups.GroupList {
	// 	groups = append(groups, group.Name)
	// }

	return raw_groups
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
		log.Println(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	// if err != nil {
	// 	log.Println(err)
	// }

	if FancyHandleError(err) {
		log.Print(err)
	}

	var users Group
	err = json.Unmarshal(body, &users)
	// if err != nil {
	// 	log.Println(err)
	// }

	if FancyHandleError(err) {
		log.Print(err)
	}

	return *users.Users
}

func createGroup(groupName string, conn Connection, client http.Client) {
	fmt.Println("Creating group", groupName)
	url := conn.url + "/app/rest/userGroups"

	group := Group{
		Key:  generateGroupKey(16),
		Name: groupName,
	}

	data, err := json.Marshal(group)

	if err != nil {
		panic(err)
	}

	createReq, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		log.Println(err)
	}
	createReq.Header.Add("Content-type", "application/json")
	createReq.Header.Add("Accept", "application/json")
	createReq.SetBasicAuth(conn.login, conn.password)

	resp, err := client.Do(createReq)
	if err != nil || resp.StatusCode > 300 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
		}
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

func (user User) addUserToGroup(group Group, userGroups Groups, conn Connection, client http.Client) {
	str := fmt.Sprintf("Adding user %s to group %s", user.Name, group.Name)
	fmt.Println(str)
	url := conn.url + "/app/rest/users/" + user.Username + "/groups"
	userGroups.GroupList = append(userGroups.GroupList, group)
	data, err := json.Marshal(userGroups)
	if err != nil {
		panic(err)
	}

	createReq, err := http.NewRequest("PUT", url, bytes.NewBuffer(data))
	if err != nil {
		log.Println(err)
	}
	createReq.Header.Add("Content-type", "application/json")
	createReq.Header.Add("Accept", "application/json")
	createReq.SetBasicAuth(conn.login, conn.password)

	resp, err := client.Do(createReq)
	defer resp.Body.Close()

	if err != nil || resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println(err)
		}
		fmt.Println("response Body:", string(body))
		//print("Error: Couldn't add user {} to group {}\n{}".format(user, group_name, resp.content))
	}
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

func userExist(ldapUser User, tcUsers Users) bool {
	for _, tcUser := range tcUsers.UsersList {
		if strings.ToLower(tcUser.Username) == strings.ToLower(ldapUser.Username) {
			return true
		}
	}
	return false
}

func userInGroup(currentUser User, userGroups Users) bool {
	for _, user := range userGroups.UsersList {
		if currentUser.Name == user.Name {
			return true
		}
	}
	return false
}

func groupExist(ldapGroup string, tcGroups Groups) bool {
	for _, tcGroup := range tcGroups.GroupList {
		if tcGroup.Name == ldapGroup {
			return true
		}
	}
	return false
}

func findTCgroup(groupName string, tcGroups Groups) Group {
	var dah Group
	for _, group := range tcGroups.GroupList {
		if group.Name == groupName {
			return group
		}
	}
	return dah
}

func generateGroupKey(n int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	letterBytes := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(16)]
	}
	return string(b)
}

func HandleError(err error) (b bool) {
	if err != nil {
		// notice that we're using 1, so it will actually log where
		// the error happened, 0 = this function, we don't want that.
		_, fn, line, _ := runtime.Caller(1)
		log.Printf("[error] %s:%d %v", fn, line, err)
		b = true
	}
	return
}

//this logs the function name as well.
func FancyHandleError(err error) (b bool) {
	if err != nil {
		// notice that we're using 1, so it will actually log the where
		// the error happened, 0 = this function, we don't want that.
		pc, fn, line, _ := runtime.Caller(1)

		log.Printf("[error] in %s[%s:%d] %v", runtime.FuncForPC(pc).Name(), fn, line, err)
		b = true
	}
	return
}

func main() {
	username := flag.String("username", "username@domain.com", "Domain login for auth")
	password := flag.String("password", "topSecret", "Password for auth")
	server := flag.String("server", "domain.com", "Address of LDAP server")
	tcServer := flag.String("tcServer", "https://teamcity.domain.com", "Address of LDAP server")
	port := flag.String("port", "389", "Port of LDAP server")
	tcUser := flag.String("tcUser", "", "User for TC with admin permissions")
	tcPassword := flag.String("tcPassword", "", "User for TC with admin permissions")
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

	client := &http.Client{}
	connection := Connection{*tcServer, *tcUser, *tcPassword}

	// использовать функцию только если есть флаг WILDCARD

	// TODO: Возвращать map из getGroupDN make(map[string]string == groupDN:grou)
	// _, groupCNs := getGroupDN("*Zabbix*", "dc=ptsecurity,dc=ru", l) // добавить выбор группы, base брать из лдап конекшн

	// // ldapUsers := getLDAPUsers(groupDN, "dc=ptsecurity,dc=ru", l) // base брать из лдап конекшн
	// // tcUsers := getTCUsers(connection, *client)

	// // userList := getTCUsers(connection, *client)

	// fish := getTCUsers(connection, *client).UsersList[0]
	// // fish.createUser(connection, *client, wg)
	// fmt.Println(fish)
	// wg := &sync.WaitGroup{}

	// for _, ldapUser := range ldapUsers {
	// 	if !userExist(ldapUser, tcUsers) {
	// 		wg.Add(1)
	// 		go ldapUser.createUser(connection, *client, wg)
	// 	}
	// }
	// wg.Wait()

	// fGroup := fish.getUserGroups(connection, *client)
	// myh := fGroup.GroupList[0]
	// fmt.Println(myh.Name)
	// //myh.getUsersFromGroup(connection, *client)
	// tcGroups := getTCGroups(connection, *client)
	// for _, tcGroup := range tcGroups.GroupList {
	// 	userGroups := fish.getUserGroups(connection, *client)
	// 	if !userInGroup(tcGroup, userGroups) {
	// 		fish.addUserToGroup(tcGroup, userGroups, connection, *client)
	// 	}
	// }
	// fishGroup := fish.getUserGroups(connection, *client)
	// myh1 := fishGroup.GroupList
	// for _, name := range myh1 {
	// 	fmt.Println(name.Name)
	// }
	// for _, ldapGroup := range groupCNs {
	// 	if !exist(ldapGroup, tcGroups) {
	// createGroup(ldapGroup, connection, *client)
	// 	}
	// }

	ldapGroups := getGroupDN("*Teamcity*", "dc=ptsecurity,dc=ru", l) // добавить выбор группы, base брать из лдап конекшн

	defer fmt.Println("\nDone")

	for groupName, groupDN := range ldapGroups {

		// // if self.ldap_object.group_exist( groupDN): ### проверяем, что группа существует в АД
		fmt.Printf("Syncing group: %s\n", groupName)

		// Create group if not exist
		tcGroups := getTCGroups(connection, *client)
		if !groupExist(groupName, tcGroups) {
			createGroup(groupName, connection, *client)
			tcGroups = getTCGroups(connection, *client)
		}

		// Create user if not exist
		ldapUsers := getLDAPUsers(groupDN, "dc=ptsecurity,dc=ru", l) // base брать из лдап конекшн
		tcUsers := getTCUsers(connection, *client)
		wg := &sync.WaitGroup{}
		for _, ldapUser := range ldapUsers {
			if !userExist(ldapUser, tcUsers) {
				wg.Add(1)
				go ldapUser.createUser(connection, *client, wg)
			}
		}

		// Get users from TC group
		currеntGroup := findTCgroup(groupName, tcGroups)
		tcGroupUsers := currеntGroup.getUsersFromGroup(connection, *client)

		// Add users to TC group
		for _, ldapUser := range ldapUsers {
			userGroups := ldapUser.getUserGroups(connection, *client)

			if !userInGroup(ldapUser, tcGroupUsers) {
				ldapUser.addUserToGroup(currеntGroup, userGroups, connection, *client)
			}
		}
	}

}
