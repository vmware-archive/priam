package main

import (
	"code.google.com/p/gopass"
	"fmt"
	"github.com/codegangsta/cli"
	"net/url"
	"strconv"
)

const coreSchemaURN = "urn:scim:schemas:core:1.0"

type dispValue struct {
	Display, Value string `json:",omitempty"`
}

type nameAttr struct {
	GivenName, FamilyName string `json:",omitempty"`
}

type userAccount struct {
	Schemas               []string                                                   `json:",omitempty"`
	UserName              string                                                     `json:",omitempty"`
	Id                    string                                                     `json:",omitempty"`
	Active                bool                                                       `json:",omitempty"`
	Emails, Groups, Roles []dispValue                                                `json:",omitempty"`
	Meta                  *struct{ Created, LastModified, Location, Version string } `json:",omitempty"`
	Name                  *nameAttr                                                  `json:",omitempty"`
	WksExt                *struct{ InternalUserType, UserStatus string }             `json:"urn:scim:schemas:extension:workspace:1.0,omitempty"`
	Password              string                                                     `json:",omitempty"`
}

func (a *userAccount) pp() {
	log(linfo, "%v, %v %v, %v\n%v\n\n", a.UserName, a.Name.GivenName, a.Name.FamilyName,
		a.Emails[0].Value, a.Id)
}

type memberValue struct {
	Value, Type, Operation string `json:",omitempty"`
}

type memberPatch struct {
	Schemas []string      `json:",omitempty"`
	Members []memberValue `json:",omitempty"`
}

type basicUser struct {
	Name, Given, Family, Email, Pwd string `yaml:",omitempty,flow"`
}

func scimGetByName(resType, nameAttr, name, authHdr string) (item map[string]interface{}, err error) {
	output := struct {
		Resources                              []map[string]interface{}
		ItemsPerPage, TotalResults, StartIndex uint
		Schemas                                []string
	}{}
	vals := url.Values{"count": {"10000"}, "filter": {fmt.Sprintf("%s eq \"%s\"", nameAttr, name)}}
	path := fmt.Sprintf("scim/%v?%v", resType, vals.Encode())
	if err = httpReq("GET", tgtURL(path), InitHdrs(authHdr, ""), nil, &output); err != nil {
		return
	}
	for _, v := range output.Resources {
		if s, ok := v[nameAttr].(string); ok && s == name {
			if item != nil {
				return nil, fmt.Errorf("multiple %v found named \"%s\"", resType, name)
			} else {
				item = v
			}
		}
	}
	if item == nil {
		err = fmt.Errorf("no %v found named \"%s\"", resType, name)
	}
	return
}

func scimNameToID(resType, nameAttr, name, authHdr string) (string, error) {
	if item, err := scimGetByName(resType, nameAttr, name, authHdr); err != nil {
		return "", err
	} else if id, ok := item["id"].(string); !ok {
		return "", fmt.Errorf("no id returned for \"%s\"", name)
	} else {
		return id, nil
	}
}

func scimList(c *cli.Context, resType string, summaryLabels ...string) {
	count, filter, vals := c.Int("count"), c.String("filter"), url.Values{}
	if count > 0 {
		vals.Set("filter", strconv.Itoa(count))
	}
	if filter != "" {
		vals.Set("filter", filter)
	}
	path := fmt.Sprintf("scim/%s?%v", resType, vals.Encode())
	if authHdr := authHeader(); authHdr != "" {
		output := make(map[string]interface{})
		if err := httpReq("GET", tgtURL(path), InitHdrs(authHdr), nil, &output); err != nil {
			log(lerr, "Error getting SCIM resources of type %s: %v\n", resType, err)
		} else {
			logppf(linfo, resType, output["Resources"], summaryLabels)
		}
	}
}

func scimPatch(resType, id, authHdr string, input interface{}) error {
	hdrs := InitHdrs(authHdr, "", "")
	hdrs["X-HTTP-Method-Override"] = "PATCH"
	path := fmt.Sprintf("scim/%s/%s", resType, id)
	return httpReq("POST", tgtURL(path), hdrs, input, nil)
}

func cmdNameToID(resType, nameAttr, name, authHdr string) string {
	if id, err := scimNameToID(resType, nameAttr, name, authHdr); err == nil {
		return id
	} else {
		log(lerr, "Error getting SCIM %s ID of %s: %v\n", resType, name, err)
	}
	return ""
}

func scimMember(c *cli.Context, resType, nameAttr string) {
	if args, authHdr := InitCmd(c, 2); authHdr != "" {
		rid, uid := cmdNameToID(resType, nameAttr, args[0], authHdr), cmdNameToID("Users", "userName", args[1], authHdr)
		if rid == "" || uid == "" {
			return
		}
		patch := memberPatch{Schemas: []string{coreSchemaURN}, Members: []memberValue{{Value: uid, Type: "User"}}}
		if c.Bool("delete") {
			patch.Members[0].Operation = "delete"
		}
		if err := scimPatch(resType, rid, authHdr, &patch); err != nil {
			log(lerr, "Error updating SCIM resource %s of type %s: %v\n", args[0], resType, err)
		} else {
			log(linfo, "Updated SCIM resource %s of type %s\n", args[0], resType)
		}
	}
}

func scimGet(c *cli.Context, resType, nameAttr string) {
	if args, authHdr := InitCmd(c, 1); authHdr != "" {
		if item, err := scimGetByName(resType, nameAttr, args[0], authHdr); err != nil {
			log(lerr, "Error getting SCIM resource named %s of type %s: %v\n", args[0], resType, err)
		} else {
			logpp(linfo, "", item)
		}
	}
}

func addUser(u *basicUser, authHdr string) error {
	acct := userAccount{UserName: u.Name, Schemas: []string{coreSchemaURN}}
	acct.Password = u.Pwd
	acct.Name = &nameAttr{FamilyName: stringOrDefault(u.Family, u.Name), GivenName: stringOrDefault(u.Given, u.Name)}
	acct.Emails = []dispValue{{Value: stringOrDefault(u.Email, u.Name+"@example.com")}}
	return httpReq("POST", tgtURL("scim/Users"), InitHdrs(authHdr, ""), &acct, &acct)
}

func cmdLoadUsers(c *cli.Context) {
	var newUsers []basicUser
	if args, authHdr := InitCmd(c, 1); authHdr != "" {
		if err := getYamlFile(args[0], &newUsers); err != nil {
			log(lerr, "could not read file of bulk users: %v\n", err)
		} else {
			for k, v := range newUsers {
				if err := addUser(&v, authHdr); err != nil {
					log(lerr, "Error adding user, line %d, name %s: %v\n", k+1, v.Name, err)
				} else {
					log(linfo, "added user %s\n", v.Name)
				}
			}
		}
	}
}

func getpwd(prompt string) string {
	if s, err := gopass.GetPass(prompt); err != nil {
		panic(err)
	} else {
		return s
	}
}

func getArgOrPassword(args []string) string {
	if len(args) > 1 {
		return args[1]
	}
	for {
		if pwd := getpwd("Password: "); pwd == getpwd("Password again: ") {
			return pwd
		}
		log(linfo, "Passwords didn't match. Try again.")
	}
}

func cmdAddUser(c *cli.Context) {
	if args, authHdr := InitCmd(c, 1); authHdr != "" {
		user := basicUser{Name: args[0], Given: c.String("given"), Family: c.String("family"),
			Email: c.String("email"), Pwd: getArgOrPassword(args)}
		if err := addUser(&user, authHdr); err != nil {
			log(lerr, "Error creating user: %v\n", err)
		} else {
			log(linfo, "User successfully added\n")
		}
	}
}

func scimDelete(c *cli.Context, resType, nameAttr string) {
	if args, authHdr := InitCmd(c, 1); authHdr != "" {
		if id := cmdNameToID(resType, nameAttr, args[0], authHdr); id != "" {
			path := fmt.Sprintf("scim/%s/%s", resType, id)
			if err := httpReq("DELETE", tgtURL(path), InitHdrs(authHdr, ""), nil, nil); err != nil {
				log(lerr, "Error deleting %s %s: %v\n", resType, args[0], err)
			} else {
				log(linfo, "%s \"%s\" deleted\n", resType, args[0])
			}
		}
	}
}

func cmdSetPassword(c *cli.Context) {
	if args, authHdr := InitCmd(c, 1); authHdr != "" {
		if id := cmdNameToID("Users", "userName", args[0], authHdr); id != "" {
			acct := userAccount{Schemas: []string{coreSchemaURN}, Password: getArgOrPassword(args)}
			if err := scimPatch("Users", id, authHdr, &acct); err != nil {
				log(lerr, "Error updating user: %v\n", err)
			} else {
				log(linfo, "User \"%s\" updated\n", acct.UserName)
			}
		}
	}
}
