package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/aquasecurity/esquery"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	ctx := context.Background()
	if err := TestEs(ctx); err != nil {
		panic(err)
	}
}

func TestEs(ctx context.Context) error {
	index := "organization_org_store"

	es, err := elasticsearch.NewDefaultClient()
	if err != nil {
		return errors.WithStack(err)
	}

	info, err := es.Info()
	if err != nil {
		return errors.WithStack(err)
	}
	defer info.Body.Close()
	b, err := ioutil.ReadAll(info.Body)
	if err != nil {
		return errors.WithStack(err)
	}

	fmt.Println(string(b))

	// if err := PrepareData(ctx, es, index); err != nil {
	// 	return errors.WithStack(err)
	// }

	r, err := esquery.Search().Query(esquery.Bool().Must(esquery.Term("organization_id", "1"))).Size(10).Run(es, es.Search.WithContext(ctx), es.Search.WithIndex(index))

	// r, err := es.Search(
	// 	es.Search.WithContext(ctx),
	// 	es.Search.WithIndex(index),
	// 	es.Search.WithQuery(query),
	// 	es.Search.WithFrom(0),
	// 	es.Search.WithSize(2),
	// )
	if err != nil {
		return errors.WithStack(err)
	}
	bs, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errors.WithStack(err)
	}
	logrus.Debugf("[%d] %s", r.StatusCode, string(bs))

	return nil
}

func PrepareData(ctx context.Context, es *elasticsearch.Client, index string) error {
	c := &Client{es: es, log: logrus.WithContext(ctx)}

	exist, err := c.IndexExists(ctx, index)
	if err != nil {
		return errors.WithStack(err)
	}
	if exist {
		c.es.Indices.Delete([]string{index})
		exist = false
	}
	if !exist {
		doc := `{
			"mappings": {
				"properties": {
					"organization_id": { "type": "keyword" },
					"org_store_id": { "type": "keyword" },
					"store_type": { "type": "keyword" },
					"store_id": { "type": "text" },
					"store_name": { "type": "text" },
					"status": { "type": "keyword" },
					"region_id": { "type": "keyword" },
					"location_id": { "type": "keyword" },
					"tags": { "type": "text" },
					"manager_ids": { "type": "keyword" },
					"created_at": { "type": "date" },
					"updated_at": { "type": "date" },
					"retail_category_id": { "type": "keyword" },
					"retail_type_id": { "type": "keyword" },
					"geo_id": { "type": "keyword" },
					"timezone": { "type": "keyword" },
					"register_count": { "type": "short" },
					"country_code": { "type": "keyword" },
					"country_name": { "type": "text" },
					"province_code": { "type": "keyword" },
					"province_name": { "type": "text" },
					"city_name": { "type": "text" },
					"address": { "type": "text" },
					"zip": { "type": "text" }
				}
			}
		}`
		if err := c.CreateIndex(ctx, index, []byte(doc)); err != nil {
			return errors.WithStack(err)
		}
	}

	bs, err := ioutil.ReadFile("data.json")
	if err != nil {
		return errors.WithStack(err)
	}
	var data []map[string]interface{}
	if err := json.Unmarshal(bs, &data); err != nil {
		return errors.WithStack(err)
	}

	for _, item := range data {
		id := cast.ToString(item["id"])

		b, err := json.Marshal(item)
		if err != nil {
			return errors.WithStack(err)
		}

		if err := c.SaveDocument(ctx, index, id, b); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

type Client struct {
	es  *elasticsearch.Client
	log *logrus.Entry
}

func (c *Client) IndexExists(ctx context.Context, index string) (bool, error) {
	r, err := c.es.Indices.Exists([]string{index}, c.es.Indices.Exists.WithContext(ctx))
	if err != nil {
		return false, errors.WithStack(err)
	}
	bs, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return true, errors.WithStack(err)
	}
	c.log.WithField("action", "IndexExists").Debugf("[%d] %s", r.StatusCode, string(bs))

	if r.StatusCode == http.StatusNotFound {
		return false, nil
	}
	return true, nil
}

func (c *Client) CreateIndex(ctx context.Context, index string, doc []byte) error {
	r, err := c.es.Indices.Create(index, c.es.Indices.Create.WithContext(ctx), c.es.Indices.Create.WithBody(bytes.NewReader(doc)))
	if err != nil {
		return errors.WithStack(err)
	}
	bs, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errors.WithStack(err)
	}
	c.log.WithField("action", "CreateIndex").Debugf("[%d] %s", r.StatusCode, string(bs))
	if r.StatusCode != 200 {
		return errors.New(string(bs))
	}
	return nil
}

func (c *Client) SaveDocument(ctx context.Context, index string, id string, doc []byte) error {
	r, err := c.es.Index(index, bytes.NewReader(doc), c.es.Index.WithContext(ctx), c.es.Index.WithDocumentID(id))
	if err != nil {
		return errors.WithStack(err)
	}
	bs, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errors.WithStack(err)
	}
	c.log.WithField("action", "SaveDocument").Debugf("[%d] %s", r.StatusCode, string(bs))
	if r.StatusCode != http.StatusOK && r.StatusCode != http.StatusCreated {
		return errors.New(string(bs))
	}
	return nil
}

func (c *Client) AddDocument(ctx context.Context, index string, id string, doc []byte) error {
	r, err := c.es.Create(index, id, bytes.NewReader(doc), c.es.Create.WithContext(ctx), c.es.Create.WithRefresh("true"))
	if err != nil {
		return errors.WithStack(err)
	}
	bs, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errors.WithStack(err)
	}
	c.log.WithField("action", "AddDocument").Debugf("[%d] %s", r.StatusCode, string(bs))
	if r.StatusCode != http.StatusOK && r.StatusCode != http.StatusCreated {
		return errors.New(string(bs))
	}
	return nil
}

func (c *Client) UpdateDocument(ctx context.Context, index string, id string, doc []byte) error {
	r, err := c.es.Update(index, id, bytes.NewReader(doc), c.es.Update.WithContext(ctx), c.es.Update.WithRefresh("true"))
	if err != nil {
		return errors.WithStack(err)
	}
	bs, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errors.WithStack(err)
	}
	c.log.WithField("action", "UpdateDocument").Debugf("[%d] %s", r.StatusCode, string(bs))
	if r.StatusCode != http.StatusOK {
		return errors.New(string(bs))
	}
	return nil
}
