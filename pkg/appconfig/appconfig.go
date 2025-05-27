package appconfig

import (
	"time"
)

// EtcdConfig holds etcd-related configuration
type EtcdConfig struct {
	Endpoints       []string      `kong:"help='etcd endpoints',default='http://localhost:2379'"`
	SnapshotDir     string        `kong:"help='Directory to store local snapshots',default='/var/lib/etcd/snapshots'"`
	SnapshotTimeout time.Duration `kong:"help='Timeout for snapshot operations',default='1m0s'"`
	Username        string        `kong:"help='etcd username for authentication'"`
	Password        string        `kong:"help='etcd password for authentication'"`
	CertFile        string        `kong:"help='etcd client certificate file'"`
	KeyFile         string        `kong:"help='etcd client key file'"`
	CaFile          string        `kong:"help='etcd CA certificate file'"`
}

// S3Config holds S3-related configuration
type S3Config struct {
	Region          string `kong:"help='S3 region',default='us-west-2'"`
	AccessKeyID     string `kong:"help='S3 access key ID'"`
	SecretAccessKey string `kong:"help='S3 secret access key'"`
	SessionToken    string `kong:"help='S3 session token'"`
	Prefix          string `kong:"help='S3 key prefix for snapshots'"`
	Bucket          string `kong:"help='S3 bucket name'"`
	EndpointURL     string `kong:"help='Custom S3 endpoint URL'"`
}

// RetentionPolicy holds retention policy configuration
type RetentionPolicy struct {
	KeepLast       int           `kong:"help='Keep last N snapshots',default=5"`
	KeepLastDays   int           `kong:"help='Keep snapshots for the last N days',default=7"`
	KeepLastHours  int           `kong:"help='Keep snapshots for the last N hours',default=24"`
	KeepLastWeeks  int           `kong:"help='Keep snapshots for the last N weeks',default=4"`
	KeepLastMonths int           `kong:"help='Keep snapshots for the last N months',default=3"`
	KeepLastYears  int           `kong:"help='Keep snapshots for the last N years',default=1"`
	RemoveLocal    bool          `kong:"help='Remove local snapshots after upload to S3'"`
	Timeout        time.Duration `kong:"help='Timeout for retention operations',default='5m'"`
}

// AppConfig is the top-level configuration structure for the application.
type AppConfig struct {
	Etcd   EtcdConfig      `kong:"embed,prefix='etcd-',group='ETCD'"`
	S3     S3Config        `kong:"embed,prefix='aws-',group='S3'"`
	Policy RetentionPolicy `kong:"embed,prefix='policy-',group='Retention Policy'"`
}
