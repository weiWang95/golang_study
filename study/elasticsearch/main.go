package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

type Product struct {
	Id    uint64 `json:"id"`
	Title string `json:"title"`
	Desc  string `json:"desc"`
}

func main() {
	es, err := elasticsearch.NewDefaultClient()
	if err != nil {
		panic(err)
	}

	// fmt.Println(es.Info())
	es.Indices.Create("product_v1", es.Indices.Create.WithBody())

	product := Product{Id: 1001, Title: "test title", Desc: "test desc"}
	data, _ := json.Marshal(product)

	req := esapi.IndexRequest{
		Index:      "product",
		DocumentID: strconv.FormatUint(product.Id, 10),
		Body:       bytes.NewReader(data),
		Refresh:    "true",
	}
	res, err := req.Do(context.TODO(), es)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		panic(err)
	}

	fmt.Println(r)
}
