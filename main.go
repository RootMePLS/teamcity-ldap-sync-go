package main

import "fmt"
import (
	"gopkg.in/ldap.v2"
	"log"
)

func main(){
	// No TLS, not recommended
	l, err := ldap.Dial("tcp", "")
	if err != nil {
		log.Println("Couldn't connect to LDAP server")
	}

	err = l.Bind("dmiroshnichenko@ptsecurity.com", "")
	if err != nil {
		// error in ldap bind
		log.Println(err)
	}

	searchRequest := ldap.NewSearchRequest(
		"dc=ptsecurity,dc=ru", // The base dn to search
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		"(&(objectClass=group)(cn=R.MPX.ISIM.Repos.All.Reader))", // The filter to apply
		[]string{"dn", "cn", "member"},                    // A list attributes to retrieve
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		log.Fatal(err)
	}

	for _, entry := range sr.Entries { //[0].Attributes[1].Values {
		fmt.Println(entry)
		//fmt.Println(entry.GetAttributeValues("member")[0])
		//fmt.Println(entry.Attributes[0])
		//fmt.Printf("%s: %v\n", entry.DN, entry.GetAttributeValue("member"))
	}
}
