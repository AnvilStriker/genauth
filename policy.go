package main

import (
	log "github.com/sirupsen/logrus"
)

func (ac *appContext) walkRsrcKindAppUsage(appName AppName, rsrcKind RsrcKind, rsrcKindUsage map[OperName][]RsrcName) {
	for operName, rsrcNames := range rsrcKindUsage {
		for _, rsrcName := range rsrcNames {
			log.WithFields(log.Fields{
				"app":  appName,
				"kind": rsrcKind,
				"oper": operName,
				"rsrc": rsrcName,
			}).Debug("app resource usage")

			switch rsrcKind {
			case "queues":
				ac.rpm.Add(ac.rsrcFullNames[rkQueuesTopics][rsrcName],
					ac.ru.Permissions.GetRoles(rkQueuesTopics, operName),
					ac.gsaNames[appName])
				ac.rpm.Add(ac.rsrcFullNames[rkQueuesSubscriptions][rsrcName],
					ac.ru.Permissions.GetRoles(rkQueuesSubscriptions, operName),
					ac.gsaNames[appName])
			default:
				ac.rpm.Add(ac.rsrcFullNames[rsrcKind][rsrcName],
					ac.ru.Permissions.GetRoles(rsrcKind, operName),
					ac.gsaNames[appName])
			}
		}
	}
}

func (ac *appContext) walkAppUsage(appName AppName, appUsage map[RsrcKind]map[OperName][]RsrcName) {
	for rsrcKind, rsrcKindUsage := range appUsage {
		ac.walkRsrcKindAppUsage(appName, rsrcKind, rsrcKindUsage)
	}

	// Add workload-identity role bindings to the app's GSA (as a resource). Secret squirrel stuff!!
	ac.rpm.Add(ac.rsrcFullNames[rkServiceAccounts][RsrcName(appName)],
		[]IAMRole{"roles/iam.serviceAccountTokenCreator"},
		ac.gsaNames[appName])
	ac.rpm.Add(ac.rsrcFullNames[rkServiceAccounts][RsrcName(appName)],
		[]IAMRole{"roles/iam.workloadIdentityUser"},
		makeGSAForKSAName(ac.ksaNames[appName], ac.locators))
}

func (ac *appContext) derivePolicies() error {
	for appName, appUsage := range ac.ru.Usage {
		ac.walkAppUsage(appName, appUsage)
	}
	log.WithField("ac.rpm", ac.rpm).Debug("derived policies")
	return nil
}
