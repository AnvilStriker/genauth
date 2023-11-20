package main

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"
)

// rsrcInfo is a source of template parameter values
type rsrcInfo struct {
	Project string
	Kind    string
	Name    string
	L       map[string]string
}

const (
	gsaUsernamePattern = "^[a-z](?:[-a-z0-9]{4,28}[a-z0-9])$"

	rkBuckets             RsrcKind = "buckets"
	rkQueues              RsrcKind = "queues"
	rkQueuesTopics        RsrcKind = "queues.topics"
	rkQueuesSubscriptions RsrcKind = "queues.subscriptions"
	rkServiceAccounts     RsrcKind = "serviceAccounts"
)

func saUsername(appName AppName) string {
	saUsername := string(appName) + "-sa"
	if strings.HasPrefix(saUsername, "scheduled-") {
		saUsername = strings.Replace(saUsername, "scheduled-", "s-", 1)
	}
	return saUsername
}

func makeKSAName(nsName NSName, appName AppName) KSAName {
	const ksaNameTemplate = "%s/%s" // simple enough that we don't need a Go template

	saName := saUsername(appName)
	return KSAName(fmt.Sprintf(ksaNameTemplate, nsName, saName))
}

// NOTE/TODO: Most Go templates should probably not be hardcoded, but rather should be loaded
// from a file and referenced symbolically.
//
// The issue here is really only around hardcoding user/company-specific naming conventions
// for things like GCP projects, GCS buckets, PubSub topics/subscriptions, etc. The structures
// of GCP resource and service account names are well-defined, so hardcoded templates for those
// would be fine as long they take whole project/bucket/etc. names as parameters.
//
// Also note that explicit project name prefixes ("owner keys") might not be appropriate
// in project ID templates even if they are user-defined depending on the user's projects'
// naming conventions, or lack thereof. (See related comment in makeQueueFullNames.)
const (
	gsaNameTText       = "{{ .Name }}@iam-shr-{{ .L.stage }}-{{ .L.unit }}.iam.gserviceaccount.com"
	gsaForKSANameTText = "gke-shr-{{ .L.stage }}-{{ .L.unit }}.svc.id.goog[{{ .Name }}]"
	gsaFullNameTText   = "projects/iam-shr-{{ .L.stage }}-{{ .L.unit }}/serviceAccounts/{{ .Name }}"
)

var (
	gsaNameT = template.Must(
		template.New("gsaName").Option("missingkey=error").Parse(gsaNameTText))
	gsaForKSANameT = template.Must(
		template.New("gsaForKSAName").Option("missingkey=error").Parse(gsaForKSANameTText))
	gsaFullNameT = template.Must(
		template.New("gsaFullName").Option("missingkey=error").Parse(gsaFullNameTText))
)

func makeGSAName(appName AppName, locators map[string]string) GSAName {
	gsaUsername := saUsername(appName)
	if strings.HasPrefix(gsaUsername, "scheduled-") {
		gsaUsername = strings.Replace(gsaUsername, "scheduled-", "s-", 1)
	}
	entry := log.WithFields(log.Fields{
		"appName":     appName,
		"gsaUsername": gsaUsername,
	})
	matched, err := regexp.MatchString(gsaUsernamePattern, gsaUsername)
	if err != nil {
		entry.WithError(err).Error("regexp.MatchString")
	} else if !matched {
		entry.Warn("invalid GSA username")
	}

	var b bytes.Buffer
	dot := rsrcInfo{Name: string(gsaUsername), L: locators}
	if err := gsaNameT.Execute(&b, &dot); err != nil {
		log.WithError(err).WithField("gsaUsername", gsaUsername).Fatal("gsaNameT.Execute")
	}
	return GSAName(b.String())
}

func makeGSAForKSAName(ksaName KSAName, locators map[string]string) GSAName {
	var b bytes.Buffer
	dot := rsrcInfo{Name: string(ksaName), L: locators}
	if err := gsaForKSANameT.Execute(&b, &dot); err != nil {
		log.WithError(err).WithField("ksaName", ksaName).Fatal("gsaForKSANameT.Execute")
	}
	return GSAName(b.String())
}

func makeGSAFullName(gsaName GSAName, locators map[string]string) RsrcFullName {
	var b bytes.Buffer
	dot := rsrcInfo{Name: string(gsaName), L: locators}
	if err := gsaFullNameT.Execute(&b, &dot); err != nil {
		log.WithError(err).WithField("gsaName", gsaName).Fatal("gsaFullNameT.Execute")
	}
	return RsrcFullName(b.String())
}

type RsrcFullNameMap map[RsrcKind]map[RsrcName]RsrcFullName

func newRsrcFullNameMap() RsrcFullNameMap {
	return RsrcFullNameMap{
		rkBuckets:             map[RsrcName]RsrcFullName{},
		rkQueuesTopics:        map[RsrcName]RsrcFullName{},
		rkQueuesSubscriptions: map[RsrcName]RsrcFullName{},
		rkServiceAccounts:     map[RsrcName]RsrcFullName{},
	}
}

func (m RsrcFullNameMap) get(rsrcKind RsrcKind, rsrcName RsrcName) RsrcFullName {
	return m[rsrcKind][rsrcName]
}

type rsrcFullNameEntry struct {
	rsrcKind     RsrcKind
	rsrcName     RsrcName
	rsrcFullName RsrcFullName
}

func makeRsrcFullNames(rsrcKind RsrcKind, ownerKey RsrcOwnerKey, rsrcName RsrcName, locators map[string]string) (entries []rsrcFullNameEntry) {
	switch rsrcKind {
	case rkBuckets:
		return makeBucketFullNames(ownerKey, rsrcName, locators)
	case rkQueues:
		return makeQueueFullNames(ownerKey, rsrcName, locators)
	default:
		// complain
		return nil
	}
}

const (
	bucketNameTText     = "{{ .L.company }}-{{ .Name }}-{{ .L.stage }}-{{ .L.region }}{{ .L.provider }}"
	bucketFullNameTText = "projects/_/buckets/{{ .Name }}"
)

var (
	bucketNameT = template.Must(
		template.New("bucketName").Option("missingkey=error").Parse(bucketNameTText))
	bucketFullNameT = template.Must(
		template.New("bucketFullName").Option("missingkey=error").Parse(bucketFullNameTText))
)

func makeBucketFullNames(_ RsrcOwnerKey, name RsrcName, locators map[string]string) []rsrcFullNameEntry {
	entry := log.WithField("name", name)

	var b bytes.Buffer
	dot := rsrcInfo{Name: string(name), L: locators}
	if err := bucketNameT.Execute(&b, &dot); err != nil {
		entry.WithError(err).Fatal("bucketNameT.Execute")
	}
	bucketName := b.String()

	b.Reset()
	dot.Name = bucketName
	if err := bucketFullNameT.Execute(&b, &dot); err != nil {
		entry.WithError(err).Fatal("bucketFullNameT.Execute")
	}
	bucketFullName := b.String()

	return []rsrcFullNameEntry{
		{
			rsrcKind:     rkBuckets,
			rsrcName:     name,
			rsrcFullName: RsrcFullName(bucketFullName),
		},
	}
}

const projectNameTText = "{{ .Name }}-{{ .L.stage }}-{{ .L.unit }}"
const pubsubNameTText = "{{ .Name }}.{{ .L.stage }}.{{ .L.region }}{{ .L.provider }}"
const pubsubFullNameTText = "projects/{{ .Project }}/{{ .Kind }}/{{ .Name }}"

var (
	projectNameT = template.Must(
		template.New("projectName").Option("missingkey=error").Parse(projectNameTText))
	pubsubNameT = template.Must(
		template.New("pubsubName").Option("missingkey=error").Parse(pubsubNameTText))
	pubsubFullNameT = template.Must(
		template.New("pubsubFullName").Option("missingkey=error").Parse(pubsubFullNameTText))
)

func makeQueueFullNames(owner RsrcOwnerKey, name RsrcName, locators map[string]string) []rsrcFullNameEntry {
	entry := log.WithField("name", name)

	var b bytes.Buffer
	dot := rsrcInfo{Name: string(name), L: locators}
	if err := pubsubNameT.Execute(&b, &dot); err != nil {
		entry.WithError(err).Fatal("pubsubNameT.Execute")
	}
	pubsubName := b.String()

	b.Reset()
	// NOTE: Here we are taking advantage of the fact that, under our current naming convention,
	//       the "resource owner" key is in fact the prefix of the owning project ID, and thus
	//       directly consumable by the template (or part of a template) that constructs the
	//       project ID. In a more general setting (e.g., working with legacy project IDs),
	//       this may not be the case, and some sort of lookup or other mapping might be needed
	//       to get the project ID (or the correct template for constructing the project ID).
	//       Similar observations apply to other resource types as well (e.g., buckets).
	dot.Name = string(owner)
	if err := projectNameT.Execute(&b, &dot); err != nil {
		entry.WithError(err).Fatal("projectNameT.Execute")
	}
	projectName := b.String()

	b.Reset()
	dot.Project, dot.Kind, dot.Name = projectName, "topics", pubsubName
	if err := pubsubFullNameT.Execute(&b, &dot); err != nil {
		entry.WithError(err).Fatal("pubsubFullNameT.Execute(topics)")
	}
	topicFullName := b.String()

	b.Reset()
	dot.Kind = "subscriptions"
	if err := pubsubFullNameT.Execute(&b, &dot); err != nil {
		entry.WithError(err).Fatal("pubsubFullNameT.Execute(subscriptions)")
	}
	subscriptionFullName := b.String()

	return []rsrcFullNameEntry{
		{
			rsrcKind:     rkQueuesTopics,
			rsrcName:     name,
			rsrcFullName: RsrcFullName(topicFullName),
		},
		{
			rsrcKind:     rkQueuesSubscriptions,
			rsrcName:     name,
			rsrcFullName: RsrcFullName(subscriptionFullName),
		},
	}
}
