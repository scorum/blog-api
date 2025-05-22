package domainprovider

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/core/connstring"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/findopt"
	. "gitlab.scorum.com/blog/core/domain"
)

// DomainProvider is responsible for providing domain for the given account
type DomainProvider struct {
	db *mongo.Database
}

func NewDomainProvider(conn connstring.ConnString) (*DomainProvider, error) {
	client, err := mongo.Connect(context.Background(), conn.String(), nil)
	if err != nil {
		return nil, err
	}

	return &DomainProvider{
		db: client.Database(conn.Database),
	}, nil
}

func (p *DomainProvider) GetByAccount(account string) (Domain, error) {
	var domain struct {
		Domain string `json:"domain"`
	}
	err := p.db.Collection("accounts").
		FindOne(
			context.Background(),
			bson.NewDocument(bson.EC.String("username", account)),
			findopt.Projection(bson.NewDocument(
				bson.EC.Int32("domain", 1),
			))).
		Decode(&domain)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// fallback to default domain
			return DomainCom, nil
		}
		return DomainCom, err
	}

	if IsValidDomain(domain.Domain) {
		return Domain(domain.Domain), nil
	}

	return DomainCom, nil
}
