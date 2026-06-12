# Disaster Recovery Plan

## Table of Contents

- [Overview](#overview)
- [Recovery Objectives](#recovery-objectives)
- [Disaster Scenarios](#disaster-scenarios)
- [Recovery Procedures](#recovery-procedures)
- [Backup Verification](#backup-verification)
- [Testing and Validation](#testing-and-validation)
- [Roles and Responsibilities](#roles-and-responsibilities)
- [Communication Plan](#communication-plan)
- [Post-Recovery](#post-recovery)
- [Appendices](#appendices)

## Overview

### Purpose

This Disaster Recovery Plan (DRP) provides comprehensive procedures for recovering from catastrophic failures affecting the Database Backup Utility or the databases it protects. The plan ensures business continuity and minimizes data loss in the event of:

- Hardware failures
- Software corruption
- Human error
- Natural disasters
- Cyber attacks
- Network outages

### Scope

This plan covers:
- Database Backup Utility infrastructure
- Protected database systems
- Backup storage systems
- Network infrastructure
- Associated services and dependencies

### Document Maintenance

- **Owner**: Infrastructure Team
- **Last Updated**: 2025-12-30
- **Review Frequency**: Quarterly
- **Next Review**: 2026-03-30

## Recovery Objectives

### RTO (Recovery Time Objective)

Maximum acceptable downtime for each service tier:

| Service Tier | RTO | Description |
|--------------|-----|-------------|
| Critical (Tier 1) | 1 hour | Production databases, payment systems |
| Important (Tier 2) | 4 hours | Customer-facing applications |
| Standard (Tier 3) | 24 hours | Internal tools, reporting databases |
| Low Priority (Tier 4) | 72 hours | Development, testing environments |

### RPO (Recovery Point Objective)

Maximum acceptable data loss for each service tier:

| Service Tier | RPO | Backup Frequency |
|--------------|-----|------------------|
| Critical (Tier 1) | 5 minutes | Continuous (streaming) |
| Important (Tier 2) | 15 minutes | Every 15 minutes |
| Standard (Tier 3) | 1 hour | Hourly |
| Low Priority (Tier 4) | 24 hours | Daily |

### Service Level Objectives

- **Backup Success Rate**: ≥ 99.5%
- **Backup Verification Rate**: 100% of critical backups
- **Recovery Test Success Rate**: ≥ 95%
- **Mean Time to Recovery (MTTR)**: < 2 hours for Tier 1

## Disaster Scenarios

### Scenario 1: Complete Database Server Failure

**Trigger**: Primary database server becomes unrecoverable

**Impact**:
- Production database unavailable
- Application downtime
- Customer impact

**Recovery Strategy**:
1. Provision new database server
2. Restore latest backup
3. Apply transaction logs (PITR)
4. Verify data integrity
5. Update connection strings
6. Resume operations

**Estimated RTO**: 1-2 hours
**Estimated RPO**: 5-15 minutes

---

### Scenario 2: Backup System Failure

**Trigger**: Backup utility server becomes unavailable

**Impact**:
- No new backups created
- Scheduled backups fail
- Monitoring gaps

**Recovery Strategy**:
1. Deploy backup server from template
2. Restore configuration from version control
3. Restore backup metadata database
4. Verify connectivity to all databases
5. Resume backup schedules

**Estimated RTO**: 30 minutes
**Estimated RPO**: 0 (backups already exist)

---

### Scenario 3: Storage System Corruption

**Trigger**: Backup storage becomes corrupted or unavailable

**Impact**:
- Cannot access existing backups
- Cannot store new backups
- Recovery operations impossible

**Recovery Strategy**:
1. Activate secondary storage location
2. Restore backup files from offsite replica
3. Verify backup integrity
4. Update storage configuration
5. Re-enable backup operations

**Estimated RTO**: 2-4 hours
**Estimated RPO**: Dependent on replication lag (typically < 1 hour)

---

### Scenario 4: Ransomware Attack

**Trigger**: Ransomware encrypts production databases and/or backups

**Impact**:
- Data encrypted and inaccessible
- Possible data exfiltration
- Extended downtime

**Recovery Strategy**:
1. Isolate affected systems immediately
2. Assess scope of compromise
3. Identify last known good backup (pre-infection)
4. Restore from immutable offsite backup
5. Rebuild systems from clean images
6. Implement additional security controls
7. Restore data and verify integrity

**Estimated RTO**: 4-8 hours
**Estimated RPO**: Up to 24 hours (last clean backup)

---

### Scenario 5: Accidental Data Deletion

**Trigger**: Critical data accidentally deleted from production

**Impact**:
- Data loss
- Business process disruption
- Potential compliance issues

**Recovery Strategy**:
1. Identify deletion timestamp
2. Select appropriate backup (pre-deletion)
3. Perform point-in-time recovery
4. Restore deleted data to staging
5. Verify data completeness
6. Merge restored data with production
7. Document incident

**Estimated RTO**: 1-2 hours
**Estimated RPO**: Depends on backup schedule (5 minutes to 1 hour)

---

### Scenario 6: Data Center Outage

**Trigger**: Complete data center failure (power, cooling, natural disaster)

**Impact**:
- All services unavailable
- No local access to systems
- Extended outage possible

**Recovery Strategy**:
1. Activate disaster recovery data center
2. Deploy backup utility from cloud image
3. Restore database servers from cloud backups
4. Update DNS to point to DR site
5. Verify all services operational
6. Begin failback planning once primary site available

**Estimated RTO**: 4-6 hours
**Estimated RPO**: Dependent on replication lag (typically < 15 minutes)

---

### Scenario 7: Network Partition

**Trigger**: Network connectivity lost between backup server and databases

**Impact**:
- Backups cannot be performed
- Monitoring unavailable
- Alerting delayed

**Recovery Strategy**:
1. Verify network connectivity
2. Activate backup paths (VPN, secondary networks)
3. Execute local backups on database servers
4. Copy backups to storage when connectivity restored
5. Resume normal operations

**Estimated RTO**: 30 minutes
**Estimated RPO**: Duration of outage (backups queued locally)

## Recovery Procedures

### Pre-Recovery Checklist

Before initiating any recovery:

- [ ] Identify disaster scenario and severity
- [ ] Notify incident commander and stakeholders
- [ ] Document current state (screenshots, logs)
- [ ] Assess impact and determine recovery priority
- [ ] Verify backup availability and integrity
- [ ] Establish communication channel
- [ ] Begin incident log
- [ ] Activate recovery team

### General Recovery Steps

#### Phase 1: Assessment (Target: 15 minutes)

```bash
# 1. Check backup utility status
systemctl status db-backup

# 2. Verify backup storage accessibility
db-backup storage check --all

# 3. List recent backups
db-backup list --database production_db --limit 10

# 4. Verify backup integrity
db-backup verify --backup-id <latest-backup-id>

# 5. Check database connectivity
db-backup test-connection --database production_db
```

#### Phase 2: Preparation (Target: 30 minutes)

```bash
# 1. Provision recovery environment
# For AWS:
aws ec2 run-instances \
  --image-id ami-recovery-template \
  --instance-type r5.2xlarge \
  --key-name recovery-key

# For bare metal:
# Boot from recovery media or PXE

# 2. Install backup utility
curl -L https://releases.backup.example.com/latest | bash

# 3. Configure backup utility
cat > /etc/db-backup/config.yaml <<EOF
storage:
  type: s3
  bucket: disaster-recovery-backups
  region: us-west-2
  access_key: ${DR_ACCESS_KEY}
  secret_key: ${DR_SECRET_KEY}
EOF

# 4. Verify configuration
db-backup validate-config
```

#### Phase 3: Recovery (Target: 1-2 hours)

```bash
# 1. Identify appropriate backup
BACKUP_ID=$(db-backup list \
  --database production_db \
  --before "2025-12-30 10:00:00" \
  --format json | jq -r '.[0].id')

echo "Selected backup: $BACKUP_ID"

# 2. Download backup from storage
db-backup download \
  --backup-id $BACKUP_ID \
  --output /recovery/backups/

# 3. Verify backup checksum
db-backup verify \
  --backup-id $BACKUP_ID \
  --checksum-only

# 4. Prepare database server
# Stop existing database if running
systemctl stop postgresql

# Remove old data (CAUTION!)
mv /var/lib/postgresql/data /var/lib/postgresql/data.old

# Create new data directory
mkdir -p /var/lib/postgresql/data
chown postgres:postgres /var/lib/postgresql/data

# 5. Restore database
db-backup restore \
  --backup-id $BACKUP_ID \
  --database production_db \
  --target-host localhost \
  --target-port 5432 \
  --confirm

# 6. Point-in-time recovery (if applicable)
db-backup restore \
  --backup-id $BACKUP_ID \
  --database production_db \
  --point-in-time "2025-12-30 10:30:00" \
  --confirm

# 7. Start database
systemctl start postgresql

# 8. Verify database status
psql -U postgres -c "SELECT version();"
psql -U postgres -c "SELECT pg_database_size('production_db');"
```

#### Phase 4: Verification (Target: 30 minutes)

```bash
# 1. Database connectivity
psql -U app_user -d production_db -c "SELECT 1;"

# 2. Row counts
psql -U postgres -d production_db <<EOF
SELECT
  schemaname,
  tablename,
  n_live_tup AS row_count
FROM pg_stat_user_tables
ORDER BY n_live_tup DESC
LIMIT 20;
EOF

# 3. Recent data verification
psql -U postgres -d production_db -c \
  "SELECT MAX(created_at) FROM orders;"

# 4. Application smoke tests
curl -f http://localhost:8080/health
curl -f http://localhost:8080/api/v1/orders?limit=10

# 5. Run data consistency checks
db-backup validate-restore \
  --backup-id $BACKUP_ID \
  --database production_db
```

#### Phase 5: Cutover (Target: 15 minutes)

```bash
# 1. Update DNS records (example using AWS Route53)
aws route53 change-resource-record-sets \
  --hosted-zone-id Z1234567890ABC \
  --change-batch file://dns-cutover.json

# 2. Update load balancer configuration
aws elbv2 modify-target-group \
  --target-group-arn arn:aws:elasticloadbalancing:... \
  --health-check-path /health

# 3. Update application configuration
kubectl set env deployment/app \
  DATABASE_HOST=db-recovery.internal

# 4. Restart application services
kubectl rollout restart deployment/app

# 5. Monitor for errors
kubectl logs -f deployment/app --tail=100
```

### Database-Specific Recovery

#### PostgreSQL Recovery

```bash
# Full restore
pg_restore \
  --host=localhost \
  --port=5432 \
  --username=postgres \
  --dbname=production_db \
  --jobs=4 \
  --verbose \
  /recovery/backups/production_db.dump

# Point-in-time recovery
# 1. Restore base backup
pg_basebackup \
  --host=localhost \
  --pgdata=/var/lib/postgresql/data \
  --wal-method=stream

# 2. Configure recovery
cat > /var/lib/postgresql/data/recovery.conf <<EOF
restore_command = 'cp /archive/wal/%f %p'
recovery_target_time = '2025-12-30 10:30:00'
EOF

# 3. Start PostgreSQL
systemctl start postgresql

# 4. Monitor recovery
tail -f /var/log/postgresql/postgresql-*.log
```

#### MySQL Recovery

```bash
# Full restore
mysql -u root -p production_db < /recovery/backups/production_db.sql

# Point-in-time recovery
# 1. Restore full backup
mysql -u root -p production_db < /recovery/backups/full_backup.sql

# 2. Apply binary logs up to specific point
mysqlbinlog \
  --stop-datetime="2025-12-30 10:30:00" \
  /var/log/mysql/mysql-bin.* | mysql -u root -p production_db

# 3. Verify recovery
mysql -u root -p -e "USE production_db; SELECT NOW();"
```

#### MongoDB Recovery

```bash
# Full restore
mongorestore \
  --host=localhost:27017 \
  --username=admin \
  --password=secret \
  --authenticationDatabase=admin \
  --db=production_db \
  --drop \
  /recovery/backups/production_db

# Point-in-time recovery
# 1. Restore base backup
mongorestore --host=localhost --db=production_db /recovery/base/

# 2. Replay oplog
mongorestore \
  --host=localhost \
  --oplogReplay \
  --oplogLimit="1735557000:1" \
  /recovery/oplog/
```

## Backup Verification

### Automated Verification

Run daily verification jobs:

```bash
# Verify all backups from last 24 hours
db-backup verify \
  --since "24 hours ago" \
  --parallel 4

# Deep verification (sample data restore)
db-backup verify \
  --backup-id latest \
  --deep \
  --sample-size 1000
```

### Manual Verification

Monthly manual verification procedure:

```bash
# 1. Select random production backup
BACKUP_ID=$(db-backup list --limit 30 --format json | \
  jq -r '.[].id' | shuf -n 1)

# 2. Provision test environment
# (Use Infrastructure as Code)

# 3. Restore to test environment
db-backup restore \
  --backup-id $BACKUP_ID \
  --target-host test-db.internal \
  --confirm

# 4. Run validation queries
psql -h test-db.internal -c "SELECT COUNT(*) FROM users;"
psql -h test-db.internal -c "SELECT MAX(created_at) FROM orders;"

# 5. Document results
cat > verification-report.md <<EOF
# Backup Verification Report

- Date: $(date)
- Backup ID: $BACKUP_ID
- Database: production_db
- Status: SUCCESS
- Notes: All data verified successfully
EOF
```

### Verification Checklist

- [ ] Backup file integrity (checksum)
- [ ] Backup file accessibility
- [ ] Decompression successful
- [ ] Decryption successful (if encrypted)
- [ ] Database restore completes
- [ ] Data completeness verified
- [ ] Schema integrity verified
- [ ] Constraints and indexes validated
- [ ] Sample data queries succeed
- [ ] Performance acceptable
- [ ] Documentation updated

## Testing and Validation

### Test Schedule

| Test Type | Frequency | Scope | Owner |
|-----------|-----------|-------|-------|
| Backup Verification | Daily | All backups | Automated |
| Recovery Simulation | Monthly | One database | Infrastructure Team |
| Full DR Test | Quarterly | All critical databases | All teams |
| Tabletop Exercise | Bi-annually | All scenarios | Leadership |
| DR Plan Review | Quarterly | Documentation | Infrastructure Team |

### Monthly Recovery Simulation

```bash
#!/bin/bash
# Monthly DR Test Script

# Configuration
TEST_DATE=$(date +%Y-%m-%d)
TEST_LOG="/var/log/dr-tests/test-${TEST_DATE}.log"
TEST_ENV="dr-test"

echo "Starting DR test: $TEST_DATE" | tee -a $TEST_LOG

# 1. Provision test environment
echo "Provisioning test environment..." | tee -a $TEST_LOG
terraform apply -auto-approve \
  -var="environment=$TEST_ENV" \
  infra/dr-test/ 2>&1 | tee -a $TEST_LOG

# 2. Select backup
BACKUP_ID=$(db-backup list --database production_db --limit 1 --format json | \
  jq -r '.[0].id')
echo "Selected backup: $BACKUP_ID" | tee -a $TEST_LOG

# 3. Restore
START_TIME=$(date +%s)
db-backup restore \
  --backup-id $BACKUP_ID \
  --target-host $TEST_ENV-db.internal \
  --confirm 2>&1 | tee -a $TEST_LOG
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo "Restore completed in ${DURATION}s" | tee -a $TEST_LOG

# 4. Validation
echo "Running validation..." | tee -a $TEST_LOG
psql -h $TEST_ENV-db.internal -c "SELECT COUNT(*) FROM users;" | tee -a $TEST_LOG

# 5. Cleanup
echo "Cleaning up..." | tee -a $TEST_LOG
terraform destroy -auto-approve \
  -var="environment=$TEST_ENV" \
  infra/dr-test/ 2>&1 | tee -a $TEST_LOG

# 6. Report
echo "Test completed successfully!" | tee -a $TEST_LOG
echo "RTO achieved: ${DURATION}s" | tee -a $TEST_LOG
```

### Quarterly Full DR Test

**Objectives**:
- Validate complete disaster recovery process
- Measure actual RTO/RPO
- Identify gaps and improvements
- Train team members

**Procedure**:

1. **T-7 days**: Schedule DR test, notify all teams
2. **T-3 days**: Prepare test environment
3. **T-1 day**: Final preparation, confirm readiness
4. **T=0**: Initiate DR test
   - Simulate disaster (graceful shutdown)
   - Activate DR procedures
   - Recover all critical services
   - Measure RTO/RPO
5. **T+1 hour**: Complete recovery
6. **T+2 hours**: Validation complete
7. **T+1 day**: Retrospective meeting
8. **T+3 days**: Action items documented

## Roles and Responsibilities

### Incident Commander

**Primary**: Director of Infrastructure
**Backup**: Senior DevOps Engineer

**Responsibilities**:
- Declare disaster and activate DR plan
- Coordinate recovery efforts
- Make critical decisions
- Communicate with executive team
- Authorize resource expenditure

### Recovery Team Lead

**Primary**: Lead Database Administrator
**Backup**: Senior DBA

**Responsibilities**:
- Execute recovery procedures
- Coordinate with team members
- Monitor recovery progress
- Escalate issues to Incident Commander
- Document recovery steps

### Database Administrators

**Team Members**: DBA Team (4 members)

**Responsibilities**:
- Execute database restore procedures
- Verify data integrity
- Perform point-in-time recovery
- Monitor database health
- Document issues and resolutions

### Infrastructure Engineers

**Team Members**: DevOps Team (6 members)

**Responsibilities**:
- Provision recovery infrastructure
- Configure networking
- Deploy backup utility
- Manage storage systems
- Monitor system performance

### Application Team

**Team Members**: Development Team (10 members)

**Responsibilities**:
- Update application configurations
- Deploy application services
- Perform application testing
- Verify business functionality
- Support user acceptance testing

### Communications Lead

**Primary**: VP of Engineering
**Backup**: Engineering Manager

**Responsibilities**:
- Notify stakeholders
- Provide status updates
- Manage external communications
- Coordinate with support team
- Post-incident communication

## Communication Plan

### Internal Communication

**Incident Declaration**:
```
SUBJECT: [INCIDENT] Disaster Recovery Activated - Production Database

SEVERITY: P1 - Critical
START TIME: 2025-12-30 10:00:00 UTC
IMPACT: Production database unavailable
ESTIMATED RTO: 2 hours

INCIDENT COMMANDER: John Doe
WAR ROOM: Zoom link / Slack channel #incident-response

INITIAL ASSESSMENT:
- Primary database server failed
- Last successful backup: 2025-12-30 09:55:00 UTC
- RPO: ~5 minutes of data loss expected
- DR procedures activated

NEXT STEPS:
1. Provision recovery server (ETA: 30 minutes)
2. Restore latest backup (ETA: 1 hour)
3. Verify data integrity (ETA: 30 minutes)
4. Application cutover (ETA: 15 minutes)

UPDATES: Every 30 minutes or as major events occur
```

**Status Update Template**:
```
UPDATE #[N] - [TIME]

PROGRESS:
- [Completed action 1]
- [Completed action 2]

CURRENT STATUS:
- [Current activity]

BLOCKERS:
- [Any issues]

NEXT STEPS:
- [Next planned actions]

REVISED ETA: [Updated estimate]
```

**Resolution Notification**:
```
SUBJECT: [RESOLVED] Disaster Recovery Complete

RESOLUTION TIME: 2025-12-30 12:15:00 UTC
TOTAL DOWNTIME: 2 hours 15 minutes
ACTUAL RPO: 3 minutes
ACTUAL RTO: 2 hours 15 minutes

SUMMARY:
- Recovery completed successfully
- All services operational
- Data integrity verified
- No data loss confirmed

POST-INCIDENT ACTIVITIES:
- Root cause analysis: 2025-12-31
- Retrospective meeting: 2026-01-02
- Action items to be documented

Thank you to all team members involved in the recovery.
```

### External Communication

**Customer Notification** (if applicable):

```
We are currently experiencing a service disruption affecting [service].
Our team is actively working on restoration.

Estimated Resolution: [Time]
Next Update: [Time]

Status Page: https://status.example.com
```

### Communication Channels

| Channel | Purpose | Audience |
|---------|---------|----------|
| Slack #incident-response | Real-time coordination | Recovery team |
| Email distribution list | Formal notifications | All engineering |
| Status page | Public updates | Customers |
| Zoom war room | Voice coordination | Recovery team |
| PagerDuty | Alerting and escalation | On-call team |

## Post-Recovery

### Immediate Actions (Within 24 hours)

- [ ] Verify all services operational
- [ ] Monitor for anomalies
- [ ] Document timeline of events
- [ ] Collect logs and evidence
- [ ] Thank recovery team
- [ ] Brief executive team

### Short-term Actions (Within 1 week)

- [ ] Conduct root cause analysis
- [ ] Hold retrospective meeting
- [ ] Document lessons learned
- [ ] Identify action items
- [ ] Update DR plan if needed
- [ ] Restore normal backup schedules

### Long-term Actions (Within 1 month)

- [ ] Implement preventive measures
- [ ] Complete action items
- [ ] Update documentation
- [ ] Review and update RTO/RPO targets
- [ ] Schedule follow-up DR test
- [ ] Report to leadership

### Retrospective Template

```markdown
# Disaster Recovery Retrospective

**Date**: 2025-12-30
**Incident**: Database server failure
**Duration**: 2 hours 15 minutes

## Timeline

| Time | Event |
|------|-------|
| 10:00 | Primary database server failed |
| 10:05 | Incident declared |
| 10:15 | Recovery team assembled |
| 10:30 | Recovery server provisioned |
| 11:00 | Backup restore started |
| 11:45 | Restore completed |
| 12:00 | Verification completed |
| 12:15 | Services restored |

## What Went Well

- Recovery procedures were clear and effective
- Team responded quickly
- Backups were accessible and valid
- Communication was timely and accurate
- RTO target was met

## What Didn't Go Well

- Initial server failure was not detected immediately
- Some documentation was outdated
- DNS propagation took longer than expected
- One team member was unreachable initially

## Action Items

1. Implement enhanced monitoring for early failure detection
2. Update documentation (specifically section X.Y)
3. Improve DNS cutover procedure
4. Establish backup contact methods for team members
5. Add automated health checks post-recovery

## Metrics

- **Target RTO**: 2 hours
- **Actual RTO**: 2 hours 15 minutes
- **Target RPO**: 5 minutes
- **Actual RPO**: 3 minutes
- **Team Response Time**: 5 minutes
```

## Appendices

### Appendix A: Emergency Contacts

| Role | Primary | Phone | Email | Backup | Phone | Email |
|------|---------|-------|-------|--------|-------|-------|
| Incident Commander | John Doe | +1-555-0101 | john@example.com | Jane Smith | +1-555-0102 | jane@example.com |
| Recovery Lead | Alice Johnson | +1-555-0103 | alice@example.com | Bob Wilson | +1-555-0104 | bob@example.com |
| DBA Team Lead | Carol Davis | +1-555-0105 | carol@example.com | Dan Brown | +1-555-0106 | dan@example.com |
| DevOps Lead | Eve Martinez | +1-555-0107 | eve@example.com | Frank Lee | +1-555-0108 | frank@example.com |

### Appendix B: Critical Systems Inventory

| System | Tier | RTO | RPO | Backup Location | Recovery Procedure |
|--------|------|-----|-----|-----------------|-------------------|
| production_db | 1 | 1h | 5m | S3 + DR site | Procedure #1 |
| customer_db | 1 | 1h | 5m | S3 + DR site | Procedure #1 |
| analytics_db | 2 | 4h | 15m | S3 | Procedure #2 |
| reporting_db | 3 | 24h | 1h | S3 | Procedure #3 |

### Appendix C: Vendor Contacts

| Vendor | Service | Support Phone | Account Number | SLA |
|--------|---------|---------------|----------------|-----|
| AWS | Cloud Infrastructure | 1-800-123-4567 | 123456789 | 24/7 |
| MongoDB Inc | Database Support | 1-800-234-5678 | MB-987654 | 24/7 |
| DataDog | Monitoring | 1-800-345-6789 | DD-456789 | Business hours |

### Appendix D: Backup Configuration

```yaml
# Production Backup Configuration
databases:
  - name: production_db
    type: postgres
    host: prod-db-01.internal
    port: 5432
    tier: 1
    backup:
      frequency: "*/5 * * * *"  # Every 5 minutes
      retention:
        daily: 7
        weekly: 4
        monthly: 12
        yearly: 5
      storage:
        - type: s3
          bucket: prod-backups-primary
          region: us-east-1
        - type: s3
          bucket: prod-backups-dr
          region: us-west-2
      compression: zstd
      encryption: true
      verify: true
```

### Appendix E: Recovery Checklists

See separate recovery procedure documents:
- [PostgreSQL Recovery Checklist](./procedures/postgres-recovery.md)
- [MySQL Recovery Checklist](./procedures/mysql-recovery.md)
- [MongoDB Recovery Checklist](./procedures/mongodb-recovery.md)

### Appendix F: Compliance Requirements

- **SOC 2**: Backup retention, encryption, access controls
- **GDPR**: Data protection, right to be forgotten
- **HIPAA**: Encryption, audit trails, access controls
- **PCI DSS**: Secure backup storage, encryption

### Appendix G: Change Log

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-30 | 1.0 | Initial version | Infrastructure Team |
| 2026-01-15 | 1.1 | Added MongoDB procedures | DBA Team |
| 2026-03-30 | 1.2 | Updated RTO/RPO targets | Infrastructure Team |

---

**Document Owner**: Infrastructure Team
**Last Reviewed**: 2025-12-30
**Next Review**: 2026-03-30
**Classification**: Internal - Confidential
