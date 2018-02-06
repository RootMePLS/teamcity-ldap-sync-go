package main

import "fmt"
import (
	"gopkg.in/ldap.v2"
	"log"
	"flag"
)

type User struct {
	username string
	name     string
	mail     string
}

func getUserAttributes(dn, baseDN string, link ldap.Conn) User {
	var user User

	filter := fmt.Sprintf("(distinguishedName=%s)", dn)
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

	user.name = fullName
	user.username = sr.Entries[0].GetAttributeValue("sAMAccountName")
	user.mail = sr.Entries[0].GetAttributeValue("mail")

	return user
}

func getUsersAD(groupName, baseDN string, link ldap.Conn) {

	var userList []User

	filter := fmt.Sprintf("(&(objectClass=group)(cn=%s))", groupName)
	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		filter,
		[]string{"sn", "givenName", "mail", "member"},
		nil,
	)

	sr, err := link.Search(searchRequest)
	if err != nil {
		log.Fatal(err)
	}

	for _, login := range sr.Entries[0].GetAttributeValues("member"){
		userList = append(userList, getUserAttributes(login, baseDN, link))
	}
	fmt.Println(userList)
}
func getUsersTC() {}

func main() {


	username := flag.String("username", "username@domain.com", "Domain login for auth")
	password := flag.String("password", "topSecret", "Password for auth")
	server := flag.String("server", "domain.com", "Address of LDAP server")
	port := flag.String("port", "389", "Port of LDAP server")

	flag.Parse()

	// No TLS, not recommended
	l, err := ldap.Dial("tcp", fmt.Sprintf("%s:%s", *server, *port))
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	err = l.Bind(*username, *password)
	if err != nil {
		// error in ldap bind
		log.Println(err)
	}

	group := "R.MPX.ISIM.Repos.All.Reader"

	//group := "DevOps"
	//filter := fmt.Sprintf("(&(objectClass=user)(memberof:1.2.840.113556.1.4.1941:=%s))", "*Zabbix*")
	//filter := fmt.Sprintf("(&(objectClass=user)(objectCategory=Person)(memberOf:1.2.840.113556.1.4.1941:=%s))", group)
	//filter := fmt.Sprintf("(distinguishedName=%s)", "CN=Vasiliy Zvyagintsev,OU=Users,OU=Tmsk,DC=ptsecurity,DC=ru")
	//filter := fmt.Sprintf("(sAMAccountName=%s)", dn)
	filter := fmt.Sprintf("(&(objectClass=group)(cn=%s))", group)
	searchRequest := ldap.NewSearchRequest(
		"dc=ptsecurity,dc=ru", // The base dn to search
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		filter,                        // The filter to apply
		[]string{"dn", "cn", "member", "sn", "givenName", "mail"}, // A list attributes to retrieve
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		log.Fatal(err)
	}

	for _, entry := range sr.Entries { // [0].Attributes[1].Values {
		fmt.Println(entry.GetAttributeValues("member"))
		fmt.Println(entry.GetAttributeValues("mail"))
		fmt.Println(entry.GetAttributeValues("dn"))
		fmt.Println(entry.GetAttributeValues("cn"))
		fmt.Println(entry.GetAttributeValues("sn"))
		fmt.Println(entry)
		//fmt.Println(entry.GetAttributeValues("member")[0])
		//fmt.Println(entry.Attributes[0])
		//fmt.Printf("%s: %v\n", entry.DN, entry.GetAttributeValue("member"))
	}


	//fmt.Println(getUserAttributes("dmiroshnichenko", "dc=ptsecurity,dc=ru", *l))
	//getUsersAD("*Zabbix*", "dc=ptsecurity,dc=ru", *l)

}
