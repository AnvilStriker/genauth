package main

import (
	log "github.com/sirupsen/logrus"
)

func (ac *appContext) deriveKSANames() error {
	for nsName, appNames := range ac.apps {
		for _, appName := range appNames {
			ac.ksaNames[appName] = makeKSAName(nsName, appName)
		}
	}
	return nil
}

func (ac *appContext) deriveGSANames() error {
	for _, appNames := range ac.apps {
		for _, appName := range appNames {
			ac.gsaNames[appName] = makeGSAName(appName, ac.locators)
		}
	}
	return nil
}

func (ac *appContext) deriveRsrcFullNames() error {
	for rsrcKind, ownedBy := range ac.ru.Resources {
		for ownerKey, rsrcNames := range ownedBy {
			for _, rsrcName := range rsrcNames {
				entries := makeRsrcFullNames(rsrcKind, ownerKey, rsrcName, ac.locators)
				for _, e := range entries {
					ac.rsrcFullNames[e.rsrcKind][e.rsrcName] = e.rsrcFullName
				}
			}
		}
	}

	// "Secretly" also make each app's service account a resource in its own right.
	// The app's service account will need "roles/iam.serviceAccountTokenCreator" on itself.
	// The "proxy" service account "gke-shr-<stage>-<unit>.svc.id.goog[<app-ns>/<app-sa-username>]"
	// will need "roles/iam.workloadIdentityUser" on the app's servvice account.
	for appName, saName := range ac.gsaNames {
		ac.rsrcFullNames["serviceAccounts"][RsrcName(appName)] = makeGSAFullName(saName, ac.locators)
	}
	return nil
}

func (ac *appContext) deriveNames() error {
	if err := ac.deriveKSANames(); err != nil {
		return err
	}
	log.WithField("ac.ksaNames", ac.ksaNames).Debug("derived KSA names")

	if err := ac.deriveGSANames(); err != nil {
		return err
	}
	log.WithField("ac.gsaNames", ac.gsaNames).Debug("derived GSA names")

	if err := ac.deriveRsrcFullNames(); err != nil {
		return err
	}
	log.WithField("ac.rsrcFullNames", ac.rsrcFullNames).Debug("derived resource full names")

	return nil
}
