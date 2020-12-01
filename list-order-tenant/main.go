package main

import (
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

	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

type ResponseGeneral struct {
	Success bool        `json:"success"`
	Rows    []SubOrders `json:"orders" bson:"orders"`
}

type SubOrders struct {
	IDDetailOrder  int     `json:"id_detail_order"`
	IDOrder        int     `json:"id_order"`
	IDProduct      int     `json:"id_product"`
	NameProduct    string  `json:"name_product"`
	Quantity       float32 `json:"quantity"`
	PricePZ        float32 `json:"price_pz"`
	PriceKG        float32 `json:"price_kg"`
	Subtotal       float32 `json:"subtotal"`
	StatusSuborder int     `json:"status_suborder"`
	IDMarket       int     `json:"id_market"`
	TypeMarket     string  `json:"type_market"`
	NameMarket     string  `json:"name_market"`
}

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"msg,omitempty" bson:"msg"`
}

func returnArrayBytes(response Response) []byte {
	res, err := json.Marshal(response)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	return res
}

func returnApiGateway(r Response, st int) (events.APIGatewayProxyResponse, error) {
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

func returnArrayBytesGeneral(response ResponseGeneral) []byte {
	res, err := json.Marshal(response)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	return res
}

func returnApiGatewayGeneral(r ResponseGeneral, st int) (events.APIGatewayProxyResponse, error) {
	resInBytes := returnArrayBytesGeneral(r)
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
		var response Response
		response.Success = true
		return returnApiGateway(response, 200)
	}

	er := getAndCheckToken(request)
	if er != nil {
		var response Response
		response.Success = false
		response.Message = "Se requiere autenticación y/o token invalido"
		return returnApiGateway(response, 401)
	}

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
		results, err := db.Query("SELECT a.id, a.id_order, a.id_product, b.name, a.quantity, a.price_pz, a.price_kg, a.total, a.status_suborder, b.id_market, c.type_market, c.name FROM detail_order as a left join products as b on a.id_product = b.id left join cat_markets as c on b.id_market = c.id WHERE a.id_tenant = ? ORDER BY a.id ASC", request.PathParameters["id_tenant"])
		if err != nil {
			panic(err)
		}

		for results.Next() {
			var row SubOrders
			err = results.Scan(&row.IDDetailOrder, &row.IDOrder, &row.IDProduct, &row.NameProduct, &row.Quantity, &row.PricePZ, &row.PriceKG, &row.Subtotal, &row.StatusSuborder, &row.IDMarket, &row.TypeMarket, &row.NameMarket)
			if err != nil {
				panic(err.Error())
			}
			response.Rows = append(response.Rows, row)
		}
	}()

	wg.Wait()

	if len(response.Rows) <= 0 {
		var res Response
		res.Success = false
		res.Message = "No se encontraron datos"
		return returnApiGateway(res, 200)
	}

	response.Success = true
	return returnApiGatewayGeneral(response, 200)
}

func main() {
	lambda.Start(handler)
}
