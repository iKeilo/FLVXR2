package middleware

import (
	"strings"
	"testing"
)

func TestCheckCommercialFeatureAllowsBillingForBusinessProfile(t *testing.T) {
	restore := snapshotLicenseState()
	defer restore()

	checkParams.licenseKey = "test-license"
	globalLicenseState.valid = true
	globalLicenseState.reason = ""
	globalLicenseState.licenseProfile = "business"
	globalLicenseState.entitlements = defaultEntitlementsForProfile("business")

	if err := CheckCommercialFeature("billing"); err != nil {
		t.Fatalf("expected business profile to allow billing, got %v", err)
	}
}

func TestCheckCommercialFeatureBlocksBillingForPersonalProfile(t *testing.T) {
	restore := snapshotLicenseState()
	defer restore()

	checkParams.licenseKey = "test-license"
	globalLicenseState.valid = true
	globalLicenseState.reason = ""
	globalLicenseState.licenseProfile = "personal"
	globalLicenseState.entitlements = defaultEntitlementsForProfile("personal")

	err := CheckCommercialFeature("billing")
	if err == nil {
		t.Fatalf("expected personal profile to block billing")
	}
	if !strings.Contains(err.Error(), "billing") {
		t.Fatalf("expected billing denial message, got %q", err.Error())
	}
}

func TestGetCommercialEntitlementsFallsBackToProfileDefaults(t *testing.T) {
	restore := snapshotLicenseState()
	defer restore()

	checkParams.licenseKey = "test-license"
	globalLicenseState.valid = true
	globalLicenseState.licenseProfile = "enterprise"
	globalLicenseState.entitlements = CommercialEntitlements{}

	entitlements := GetCommercialEntitlements()
	if !entitlements.BillingAllowed {
		t.Fatalf("expected enterprise profile fallback to allow billing")
	}
	if !entitlements.MultiTenantAllowed {
		t.Fatalf("expected enterprise profile fallback to allow multi-tenant")
	}
	if entitlements.DeploymentScope == "" {
		t.Fatalf("expected enterprise profile fallback to populate deployment scope")
	}
}

func snapshotLicenseState() func() {
	globalLicenseState.mu.Lock()
	checkParams.mu.Lock()

	oldValid := globalLicenseState.valid
	oldExpireTime := globalLicenseState.expireTime
	oldReason := globalLicenseState.reason
	oldIsTrial := globalLicenseState.isTrial
	oldPlanName := globalLicenseState.planName
	oldLicenseProfile := globalLicenseState.licenseProfile
	oldEntitlements := globalLicenseState.entitlements
	oldLastCheck := globalLicenseState.LastCheck

	oldServerURL := checkParams.serverURL
	oldLicenseKey := checkParams.licenseKey
	oldDomain := checkParams.domain
	oldAccessDomain := checkParams.accessDomain
	oldAccessProtocol := checkParams.accessProtocol
	oldDomainFromConfig := checkParams.domainFromConfig

	checkParams.mu.Unlock()
	globalLicenseState.mu.Unlock()

	return func() {
		globalLicenseState.mu.Lock()
		checkParams.mu.Lock()

		globalLicenseState.valid = oldValid
		globalLicenseState.expireTime = oldExpireTime
		globalLicenseState.reason = oldReason
		globalLicenseState.isTrial = oldIsTrial
		globalLicenseState.planName = oldPlanName
		globalLicenseState.licenseProfile = oldLicenseProfile
		globalLicenseState.entitlements = oldEntitlements
		globalLicenseState.LastCheck = oldLastCheck

		checkParams.serverURL = oldServerURL
		checkParams.licenseKey = oldLicenseKey
		checkParams.domain = oldDomain
		checkParams.accessDomain = oldAccessDomain
		checkParams.accessProtocol = oldAccessProtocol
		checkParams.domainFromConfig = oldDomainFromConfig

		checkParams.mu.Unlock()
		globalLicenseState.mu.Unlock()
	}
}
