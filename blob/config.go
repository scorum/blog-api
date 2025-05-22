package blob

// Config represents blob settings
type Config struct {
	AccountName string `yaml:"account_name"`
	AccountKey  string `yaml:"account_key"`
	Container   string `yaml:"container"`
	CDNDomain   string `yaml:"cdn_domain"`
}
