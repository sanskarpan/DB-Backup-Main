package compliance.residency

# Default deny cross-border transfers
default allow = false

# EU regions for GDPR compliance
eu_regions := ["eu-west", "eu-central", "eu-north", "eu-south"]

# US regions
us_regions := ["us-east", "us-west", "us-central", "us-gov"]

# APAC regions
apac_regions := ["ap-south", "ap-northeast", "ap-southeast", "ap-east"]

# Data classifications
public_data := "public"
internal_data := "internal"
confidential_data := "confidential"
restricted_data := "restricted"

# Allow transfers within the same region
allow {
    input.source_region == input.target_region
}

# Allow transfers within EU for any data type (GDPR adequate protection)
allow {
    input.source_region in eu_regions
    input.target_region in eu_regions
}

# Allow transfers within US for non-restricted data
allow {
    input.source_region in us_regions
    input.target_region in us_regions
    input.classification != restricted_data
}

# Allow transfers within APAC for non-restricted data
allow {
    input.source_region in apac_regions
    input.target_region in apac_regions
    input.classification != restricted_data
}

# Allow public data transfers anywhere
allow {
    input.classification == public_data
}

# Allow EU to US transfers with standard contractual clauses
allow {
    input.source_region in eu_regions
    input.target_region in us_regions
    has_standard_contractual_clauses
    input.classification in [internal_data, confidential_data]
}

# Allow transfers with explicit consent
allow {
    has_user_consent
    input.classification != restricted_data
}

# Allow transfers for adequacy decisions
allow {
    has_adequacy_decision
}

# Check for standard contractual clauses
has_standard_contractual_clauses {
    input.legal_mechanisms[_] == "standard_contractual_clauses"
}

# Check for user consent
has_user_consent {
    input.legal_mechanisms[_] == "explicit_consent"
    input.consent_verified == true
}

# Check for adequacy decision
has_adequacy_decision {
    input.legal_mechanisms[_] == "adequacy_decision"
    adequacy_pair
}

adequacy_pair {
    input.source_region in eu_regions
    input.target_region in ["canada-central", "japan-east", "uk-south"]
}

# Deny restricted data transfers outside origin region
deny {
    input.classification == restricted_data
    input.source_region != input.target_region
}

# Deny EU data to non-adequate countries without safeguards
deny {
    input.source_region in eu_regions
    not input.target_region in eu_regions
    not has_gdpr_safeguards
}

has_gdpr_safeguards {
    has_standard_contractual_clauses
}

has_gdpr_safeguards {
    has_user_consent
}

has_gdpr_safeguards {
    has_adequacy_decision
}

# Deny transfers during data freeze periods
deny {
    input.data_freeze_active == true
    input.override_freeze != true
}

# Deny if target region has data localization laws
deny {
    target_has_localization_requirements
    not complies_with_localization
}

target_has_localization_requirements {
    input.target_region in ["russia", "china", "india"]
}

complies_with_localization {
    input.local_processing_enabled == true
    input.local_storage_enabled == true
}

# Specific rules for financial data
allow {
    input.data_type == "financial"
    input.source_region == input.target_region
}

deny {
    input.data_type == "financial"
    input.source_region != input.target_region
    not has_financial_data_transfer_approval
}

has_financial_data_transfer_approval {
    input.approvals[_] == "compliance_officer"
    input.approvals[_] == "legal_team"
}

# Specific rules for health data (HIPAA compliance)
deny {
    input.data_type == "health"
    input.classification == restricted_data
    not is_hipaa_compliant_transfer
}

is_hipaa_compliant_transfer {
    input.source_region in us_regions
    input.target_region in us_regions
    input.baa_signed == true # Business Associate Agreement
}

# Reasons for denial
reason = msg {
    not allow
    input.classification == restricted_data
    input.source_region != input.target_region
    msg := "Restricted data cannot be transferred outside its origin region"
}

reason = msg {
    not allow
    input.source_region in eu_regions
    not input.target_region in eu_regions
    not has_gdpr_safeguards
    msg := sprintf("Transfer from EU region %s to %s requires GDPR safeguards (SCC, consent, or adequacy decision)", [
        input.source_region,
        input.target_region
    ])
}

reason = msg {
    not allow
    input.data_freeze_active == true
    msg := "Data transfer denied: Data freeze period is active"
}

reason = msg {
    not allow
    target_has_localization_requirements
    not complies_with_localization
    msg := sprintf("Target region %s has data localization requirements that are not met", [
        input.target_region
    ])
}

reason = msg {
    not allow
    input.data_type == "financial"
    not has_financial_data_transfer_approval
    msg := "Financial data transfer requires approval from compliance officer and legal team"
}

reason = msg {
    not allow
    input.data_type == "health"
    not is_hipaa_compliant_transfer
    msg := "Health data transfer must comply with HIPAA requirements (US regions only, BAA required)"
}

# Additional details
details = {
    "source_region": input.source_region,
    "target_region": input.target_region,
    "classification": input.classification,
    "data_type": input.data_type,
    "same_region": input.source_region == input.target_region,
    "has_safeguards": has_gdpr_safeguards,
    "legal_mechanisms": input.legal_mechanisms,
}

# Recommendations for approval
recommendations = recs {
    not allow
    recs := {
        "required_mechanisms": required_legal_mechanisms,
        "alternative_regions": suggest_alternative_regions,
        "required_approvals": required_approvals,
    }
}

required_legal_mechanisms = mechanisms {
    input.source_region in eu_regions
    not input.target_region in eu_regions
    mechanisms := ["standard_contractual_clauses", "explicit_consent", "adequacy_decision"]
}

suggest_alternative_regions = regions {
    input.source_region in eu_regions
    regions := eu_regions
}

suggest_alternative_regions = regions {
    input.source_region in us_regions
    regions := us_regions
}

required_approvals = approvals {
    input.data_type == "financial"
    approvals := ["compliance_officer", "legal_team"]
}

required_approvals = approvals {
    input.classification == restricted_data
    approvals := ["data_protection_officer", "ciso", "legal_team"]
}
