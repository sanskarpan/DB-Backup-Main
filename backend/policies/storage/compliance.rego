package storage.compliance

# Default deny
default allow = false

# Allowed storage types
allowed_storage_types := ["s3", "gcs", "azure", "local"]

# Regions requiring encryption
encryption_required_regions := ["eu-west", "eu-central", "us-gov"]

# GDPR-compliant regions
gdpr_regions := ["eu-west", "eu-central", "eu-north"]

# Allow storage if all compliance rules pass
allow {
    is_valid_storage_type
    meets_encryption_requirements
    meets_regional_requirements
    meets_data_classification_requirements
}

# Check if storage type is valid
is_valid_storage_type {
    input.storage_type in allowed_storage_types
}

# Check encryption requirements
meets_encryption_requirements {
    not requires_encryption
}

meets_encryption_requirements {
    requires_encryption
    input.encrypted == true
}

requires_encryption {
    input.region in encryption_required_regions
}

requires_encryption {
    input.metadata.data_classification == "confidential"
}

requires_encryption {
    input.metadata.data_classification == "restricted"
}

# Check regional requirements
meets_regional_requirements {
    not has_regional_restrictions
}

meets_regional_requirements {
    has_regional_restrictions
    region_is_compliant
}

has_regional_restrictions {
    input.metadata.compliance_frameworks[_] == "GDPR"
}

region_is_compliant {
    input.metadata.compliance_frameworks[_] == "GDPR"
    input.region in gdpr_regions
}

# Check data classification requirements
meets_data_classification_requirements {
    input.metadata.data_classification == "public"
}

meets_data_classification_requirements {
    input.metadata.data_classification == "internal"
    input.encrypted == true
}

meets_data_classification_requirements {
    input.metadata.data_classification in ["confidential", "restricted"]
    input.encrypted == true
    input.metadata.access_logging_enabled == true
}

# Reasons for denial
reason = msg {
    not is_valid_storage_type
    msg := sprintf("Invalid storage type: %s. Allowed types: %v", [
        input.storage_type,
        allowed_storage_types
    ])
}

reason = msg {
    not meets_encryption_requirements
    requires_encryption
    msg := sprintf("Encryption required for region %s or data classification %s", [
        input.region,
        input.metadata.data_classification
    ])
}

reason = msg {
    not meets_regional_requirements
    has_regional_restrictions
    msg := sprintf("Region %s does not meet compliance requirements for %v", [
        input.region,
        input.metadata.compliance_frameworks
    ])
}

reason = msg {
    not meets_data_classification_requirements
    msg := sprintf("Storage configuration does not meet requirements for data classification: %s", [
        input.metadata.data_classification
    ])
}

# Additional checks for backup retention
check_retention_policy {
    input.metadata.retention_days >= minimum_retention_days
}

minimum_retention_days = days {
    input.metadata.compliance_frameworks[_] == "GDPR"
    days := 30
}

minimum_retention_days = days {
    input.metadata.compliance_frameworks[_] == "HIPAA"
    days := 2555 # 7 years
}

minimum_retention_days = 7 # Default
