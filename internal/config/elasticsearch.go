package config

import (
	"github.com/olivere/elastic"
)

// GetElasticsearch 获取Elasticsearch客户端
func (c *Context) GetElasticsearch() *elastic.Client {
	if c.elasticClient == nil {
		var err error
		c.elasticClient, err = elastic.NewClient(elastic.SetURL(c.GetConfig().ElasticsearchURL))
		if err != nil {
			panic(err)
		}
		return c.elasticClient
	}
	return c.elasticClient

}
