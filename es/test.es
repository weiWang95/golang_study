GET /organization_org_store/_search
{
  "from": 0,
  "query": {
    "bool": {
      "must": [
        {
          "term": {
            "organization_id": {
              "value": 1
            }
          }
        },
        {
          "multi_match": {
            "fields": [
              "store_name",
              "store_id",
              "city_name",
              "address"
            ],
            "query": "深圳"
          }
        },
        {
          "terms": {
            "status": [
              "active",
              "available",
              "expire_soon",
              "develop"
            ]
          }
        }
      ]
    }
  },
  "size": 10,
  "sort": [
    {
      "_score": { "order": "desc" }
    },
    {
      "created_at": {
        "order": "desc"
      }
    }
  ]
}

POST /organization_org_store/_bulk
{"update": {"_id": "340478776276169217","_index": "organization_org_store"}}
{"manager_ids":[350462241423766529]}
