# Casbin Go Cloud Development kit based watcher

[Casbin](https://github.com/casbin/casbin) watcher built on top of [gocloud.dev](https://gocloud.dev/).

## Installation

```
go get github.com/rusenask/casbin-go-cloud-watcher
```

## Usage

Configuration is slightly different for each provider as it needs to get different settings from environment. You can read more about URLs and configuration here: https://gocloud.dev/concepts/urls/.

Supported providers:
- [NATS](https://nats.io/)
- [GCP Cloud Pub/Sub](https://cloud.google.com/pubsub/)
- [AWS SNS](https://aws.amazon.com/sns)
- [AWS SQS](https://aws.amazon.com/sqs/)
- [Azure Service Bus](https://azure.microsoft.com/en-us/services/service-bus/)
- [Apache Kafka](https://kafka.apache.org/)
- [RabbitMQ](https://www.rabbitmq.com/)
- In memory pubsub (useful for local testing and single node installs)

You can view provider configuration examples here: https://github.com/google/go-cloud/tree/master/pubsub.

### NATS

```go
import (
    cloudwatcher "github.com/rusenask/casbin-go-cloud-watcher"
    "github.com/casbin/casbin"
)

func main() {
    // This URL will Dial the NATS server at the URL in the environment variable
	  // NATS_SERVER_URL and send messages with subject "casbin-policy-updates".
    os.Setenv("NATS_SERVER_URL", "nats://localhost:4222")
    ctx, cancel := context.WithCancel(context.Background())
	  defer cancel()
    watcher, _ := cloudwatcher.New(ctx, "nats://casbin-policy-updates")

    enforcer := casbin.NewSyncedEnforcer("model.conf", "policy.csv")
    enforcer.SetWatcher(watcher)

    watcher.SetUpdateCallback(func(m string) {
      enforcer.LoadPolicy()
    })
}
```

### In Memory

```go
import (
    cloudwatcher "github.com/rusenask/casbin-go-cloud-watcher"
    "github.com/casbin/casbin"
)

func main() {
    ctx, cancel := context.WithCancel(context.Background())
	  defer cancel()
    watcher, _ := cloudwatcher.New(ctx, "mem://topicA")

    enforcer := casbin.NewSyncedEnforcer("model.conf", "policy.csv")
    enforcer.SetWatcher(watcher)

    watcher.SetUpdateCallback(func(m string) {
      enforcer.LoadPolicy()
    })
}
```

### GCP Cloud Pub/Sub

URLs are `gcppubsub://projects/myproject/topics/mytopic`. The URLs use the project ID and the topic ID. See [Application Default Credentials](https://cloud.google.com/docs/authentication/production) to learn about authentication alternatives, including using environment variables.

```go
import (
    cloudwatcher "github.com/rusenask/casbin-go-cloud-watcher"
    "github.com/casbin/casbin"
)

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    watcher, _ := cloudwatcher.New(ctx, "gcppubsub://projects/myproject/topics/mytopic")

    enforcer := casbin.NewSyncedEnforcer("model.conf", "policy.csv")
    enforcer.SetWatcher(watcher)

    watcher.SetUpdateCallback(func(m string) {
      enforcer.LoadPolicy()
    })
}
```

### Amazon Simple Notification Service (SNS)

The watcher can publish to an [Amazon Simple Notification Service](https://aws.amazon.com/sns/) (SNS) topic. SNS URLs in the Go CDK use the Amazon Resource Name (ARN) to identify the topic. You should specify the region query parameter to ensure your application connects to the correct region.

Watcher will create a default AWS Session with the *SharedConfigEnable* option enabled; if you have authenticated with the AWS CLI, it will use those credentials. See [AWS Session](https://docs.aws.amazon.com/sdk-for-go/api/aws/session/) to learn about authentication alternatives, including using environment variables.

```go
import (
    cloudwatcher "github.com/rusenask/casbin-go-cloud-watcher"
    "github.com/casbin/casbin"
)

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    const topicARN = "arn:aws:sns:us-east-2:123456789012:mytopic"
    // Note the 3 slashes; ARNs have multiple colons and therefore aren't valid
    // as hostnames in the URL.
    watcher, _ := cloudwatcher.New(ctx, "awssns:///"+topicARN+"?region=us-east-2")

    enforcer := casbin.NewSyncedEnforcer("model.conf", "policy.csv")
    enforcer.SetWatcher(watcher)

    watcher.SetUpdateCallback(func(m string) {
      enforcer.LoadPolicy()
    })
}
```

## About Go Cloud Dev

Portable Cloud APIs in Go. Strives to implement these APIs for the leading Cloud providers: AWS, GCP and Azure, as well as provide a local (on-prem) implementation such as Kafka, NATS, etc.

Using the Go CDK you can write your application code once using these idiomatic APIs, test locally using the local versions, and then deploy to a cloud provider with only minimal setup-time changes.
