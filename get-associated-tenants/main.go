package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
)

type RequestBody struct {
	//IDCP       int `json:"id_cp" bson:"id_cp"`
	IDMarket   int    `json:"id_market" bson:"id_market"`
	TypeMarket string `json:"type_market" bson:"type_market"`
}

type ResponseGeneral struct {
	Success bool   `json:"success"`
	Message string `json:"msg,omitempty" bson:"msg"`
	Rows    []Row  `json:"row" bson:"row"`
}

type Row struct {
	IDCategory   int       `json:"id_category"`
	Name         string    `json:"name"`
	TenantsArray []Tenants `json:"tenants_array" bson:"tenants_array"`
}

//id_tenant == id_user
type Tenants struct {
	IDTenant    int    `json:"id_tenant"`
	IDTypeUser  int    `json:"id_type_user"`
	Zone        string `json:"zone" bson:"zone"`
	IDMarket    int    `json:"id_market" bson:"id_market"`
	LocalNumber string `json:"local_number"`
	NameTenant  string `json:"name_tenenant" bson:"name_tenenant"`
}

func returnArrayBytes(response ResponseGeneral) []byte {
	res, err := json.Marshal(response)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	return res
}

func returnApiGateway(r ResponseGeneral, st int) (events.APIGatewayProxyResponse, error) {
	resInBytes := returnArrayBytes(r)
	return events.APIGatewayProxyResponse{
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "*",
			"Access-Control-Allow-Headers": "authorization, content-type",
		},
		Body:       fmt.Sprintf(string(resInBytes)),
		StatusCode: st,
	}, nil
}

func decodeTokenHeader(str string) (string, error) {
	bearer := strings.Split(str, " ")
	if len(bearer) < 2 {
		return "", errors.New("Authorization header malformed")
	}
	token := bearer[1]
	return token, nil
}

func checkToken(token string) error {
	jwtKey, ok := os.LookupEnv("KEY_JWT")
	if !ok {
		fmt.Println("no existe key para jwt")
		os.Exit(1)
	}
	tok, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtKey), nil
	})
	if tok.Valid {
		fmt.Println("all ok")
		return nil
	} else if ve, ok := err.(*jwt.ValidationError); ok {
		if ve.Errors&jwt.ValidationErrorMalformed != 0 {
			fmt.Println("Malformed Token")
			return err
		} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
			fmt.Println("Token Expired")
			return err
		} else {
			fmt.Println("Could not handled this token", err)
			return err
		}
		//return err
	} else {
		fmt.Println("Could not handled this token", err)
		return err
	}
}

func getAndCheckToken(request events.APIGatewayProxyRequest) error {
	auth, ok := request.Headers["Authorization"]
	if !ok {
		//log.Println("La cabecera no presenta una Autorización")
		//os.Exit(1)
		err := errors.New("not header exist such as Bearer token")
		return err
	}

	token, erro := decodeTokenHeader(auth)
	if erro != nil {
		log.Println(erro)
		return erro
	}

	erro = checkToken(token)
	if erro != nil {
		return erro
	}

	return nil
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if request.HTTPMethod == "OPTIONS" {
		var response ResponseGeneral
		response.Success = true
		return returnApiGateway(response, 200)
	}

	er := getAndCheckToken(request)
	if er != nil {
		var response ResponseGeneral
		response.Success = false
		response.Message = "Se requiere autenticación y/o token invalido"
		return returnApiGateway(response, 401)
	}

	var bodyData *RequestBody
	err := json.Unmarshal([]byte(request.Body), &bodyData)

	userDB, ok := os.LookupEnv("USER_DB")
	if !ok {
		userDB = "test"
	}
	pwdDB, ok := os.LookupEnv("PASSWORD_DB")
	if !ok {
		pwdDB = "xxxxxxxxxxxx"
	}
	hostDB, ok := os.LookupEnv("HOST_DB")
	if !ok {
		hostDB = "localhost"
	}
	portDB, ok := os.LookupEnv("PORT_DB")
	if !ok {
		portDB = "3306"
	}
	nameDB, ok := os.LookupEnv("NAME_DB")
	if !ok {
		nameDB = "test"
	}

	//connection
	db, err := sql.Open("mysql", fmt.Sprintf("%v:%v@tcp(%v:%v)/%v",
		userDB,
		pwdDB,
		hostDB,
		portDB,
		nameDB,
	))
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	var wg sync.WaitGroup
	var response ResponseGeneral

	wg.Add(1)
	go func() {
		defer wg.Done()
		results, err := db.Query("SELECT id, name FROM category_products where type = ? and active = 1 ORDER BY id ASC", bodyData.TypeMarket)
		if err != nil {
			panic(err)
		}

		for results.Next() {
			var row Row
			err = results.Scan(&row.IDCategory, &row.Name)
			if err != nil {
				panic(err.Error())
			}
			response.Rows = append(response.Rows, row)
		}
	}()

	wg.Wait()

	for i := range response.Rows {
		//replace market por id_market, no guardar el nombre del market si no su id, modificar en la bd en todos los lados donde se relacione
		wg.Add(1)
		go func() {
			defer wg.Done()
			results, err := db.Query("SELECT a.id_user, b.id_type_user, b.zone, b.id_market, b.local, b.name FROM products as a left join users as b on a.id_user = b.id where a.id_type_category = ? and b.active = 1 and id_market = ? GROUP BY a.id_user ORDER BY a.id ASC", response.Rows[i].IDCategory, bodyData.IDMarket)
			if err != nil {
				panic(err)
			}

			for results.Next() {
				var tenant_row Tenants
				err = results.Scan(&tenant_row.IDTenant, &tenant_row.IDTypeUser, &tenant_row.Zone, &tenant_row.IDMarket, &tenant_row.LocalNumber, &tenant_row.NameTenant)
				if err != nil {
					panic(err.Error())
				}
				response.Rows[i].TenantsArray = append(response.Rows[i].TenantsArray, tenant_row)
			}
		}()

		wg.Wait()
	}

	if len(response.Rows) <= 0 {
		var res ResponseGeneral
		res.Success = false
		res.Message = "No se encontraron datos"
		return returnApiGateway(res, 200)
	}

	response.Success = true
	return returnApiGateway(response, 200)
}

func main() {
	lambda.Start(handler)
}
