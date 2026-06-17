import React from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  TouchableOpacity,
  Alert,
} from 'react-native';
import Icon from 'react-native-vector-icons/Ionicons';
import {useDispatch} from 'react-redux';
import {fetchBackups} from '../store/backupsSlice';
import {apiService} from '../services/api';

export default function BackupDetailScreen({route, navigation}: any) {
  const dispatch = useDispatch();
  const {backup} = route.params;

  const statusColors: Record<string, {bg: string; text: string; icon: string}> = {
    completed: {bg: '#d1fae5', text: '#065f46', icon: 'checkmark-circle'},
    running: {bg: '#dbeafe', text: '#1e40af', icon: 'time'},
    failed: {bg: '#fee2e2', text: '#991b1b', icon: 'close-circle'},
    pending: {bg: '#fef3c7', text: '#92400e', icon: 'hourglass'},
  };

  const statusInfo = statusColors[backup.status] || statusColors.pending;

  const formatSize = (bytes?: number) => {
    if (!bytes) return 'N/A';
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
  };

  const formatDuration = (seconds?: number) => {
    if (!seconds) return 'N/A';
    if (seconds < 60) return `${seconds}s`;
    const m = Math.floor(seconds / 60);
    const s = seconds % 60;
    return `${m}m ${s}s`;
  };

  const handleDownload = () => {
    Alert.alert(
      'Download Backup',
      'Download will start in the background.',
      [{text: 'OK'}],
    );
  };

  const handleRestore = () => {
    Alert.alert(
      'Restore Backup',
      `Are you sure you want to restore "${backup.database_name}" from this backup? This cannot be undone.`,
      [
        {text: 'Cancel', style: 'cancel'},
        {
          text: 'Restore',
          style: 'destructive',
          onPress: async () => {
            try {
              await apiService.restoreBackup(backup.id);
              Alert.alert('Success', 'Restore initiated successfully.');
            } catch (error) {
              Alert.alert('Error', 'Failed to initiate restore. Please try again.');
            }
          },
        },
      ],
    );
  };

  const handleDelete = () => {
    Alert.alert(
      'Delete Backup',
      'Are you sure you want to delete this backup? This cannot be undone.',
      [
        {text: 'Cancel', style: 'cancel'},
        {
          text: 'Delete',
          style: 'destructive',
          onPress: async () => {
            try {
              await apiService.deleteBackup(backup.id);
              await dispatch(fetchBackups() as any);
              navigation.goBack();
            } catch (error) {
              Alert.alert('Error', 'Failed to delete backup. Please try again.');
            }
          },
        },
      ],
    );
  };

  return (
    <ScrollView style={styles.container}>
      {/* Status Banner */}
      <View style={[styles.statusBanner, {backgroundColor: statusInfo.bg}]}>
        <Icon name={statusInfo.icon} size={32} color={statusInfo.text} />
        <Text style={[styles.statusText, {color: statusInfo.text}]}>
          {backup.status.charAt(0).toUpperCase() + backup.status.slice(1)}
        </Text>
      </View>

      {/* Details Card */}
      <View style={styles.card}>
        <Text style={styles.cardTitle}>Backup Details</Text>

        <DetailRow
          icon="server-outline"
          label="Database"
          value={backup.database_name || 'Unknown'}
          iconColor="#3b82f6"
        />
        <DetailRow
          icon="calendar-outline"
          label="Created"
          value={new Date(backup.created_at).toLocaleString()}
          iconColor="#6366f1"
        />
        <DetailRow
          icon="archive-outline"
          label="Size"
          value={formatSize(backup.size)}
          iconColor="#10b981"
        />
        <DetailRow
          icon="timer-outline"
          label="Duration"
          value={formatDuration(backup.duration)}
          iconColor="#f59e0b"
        />
        <DetailRow
          icon="key-outline"
          label="Backup ID"
          value={backup.id}
          iconColor="#6b7280"
          monospace
        />
        {backup.database_id ? (
          <DetailRow
            icon="link-outline"
            label="Database ID"
            value={backup.database_id}
            iconColor="#6b7280"
            monospace
          />
        ) : null}
      </View>

      {/* Actions */}
      <View style={styles.actionsCard}>
        <Text style={styles.cardTitle}>Actions</Text>

        <TouchableOpacity style={styles.actionButton} onPress={handleDownload}>
          <View style={[styles.actionIcon, {backgroundColor: '#dbeafe'}]}>
            <Icon name="download-outline" size={22} color="#1e40af" />
          </View>
          <View style={styles.actionInfo}>
            <Text style={styles.actionLabel}>Download</Text>
            <Text style={styles.actionDescription}>
              Download backup file to device
            </Text>
          </View>
          <Icon name="chevron-forward" size={18} color="#9ca3af" />
        </TouchableOpacity>

        {backup.status === 'completed' && (
          <TouchableOpacity
            style={styles.actionButton}
            onPress={handleRestore}>
            <View style={[styles.actionIcon, {backgroundColor: '#d1fae5'}]}>
              <Icon name="refresh-circle-outline" size={22} color="#065f46" />
            </View>
            <View style={styles.actionInfo}>
              <Text style={styles.actionLabel}>Restore</Text>
              <Text style={styles.actionDescription}>
                Restore database from this backup
              </Text>
            </View>
            <Icon name="chevron-forward" size={18} color="#9ca3af" />
          </TouchableOpacity>
        )}

        <TouchableOpacity
          style={[styles.actionButton, styles.deleteAction]}
          onPress={handleDelete}>
          <View style={[styles.actionIcon, {backgroundColor: '#fee2e2'}]}>
            <Icon name="trash-outline" size={22} color="#991b1b" />
          </View>
          <View style={styles.actionInfo}>
            <Text style={[styles.actionLabel, {color: '#ef4444'}]}>Delete</Text>
            <Text style={styles.actionDescription}>
              Permanently remove this backup
            </Text>
          </View>
          <Icon name="chevron-forward" size={18} color="#9ca3af" />
        </TouchableOpacity>
      </View>

      <View style={styles.bottomSpacer} />
    </ScrollView>
  );
}

function DetailRow({
  icon,
  label,
  value,
  iconColor,
  monospace,
}: {
  icon: string;
  label: string;
  value: string;
  iconColor: string;
  monospace?: boolean;
}) {
  return (
    <View style={styles.detailRow}>
      <Icon name={icon} size={18} color={iconColor} style={styles.detailIcon} />
      <Text style={styles.detailLabel}>{label}</Text>
      <Text
        style={[styles.detailValue, monospace && styles.monospace]}
        numberOfLines={1}
        ellipsizeMode="middle">
        {value}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f9fafb',
  },
  statusBanner: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    padding: 24,
    gap: 12,
  },
  statusText: {
    fontSize: 22,
    fontWeight: 'bold',
  },
  card: {
    backgroundColor: 'white',
    borderRadius: 12,
    margin: 16,
    marginBottom: 0,
    padding: 16,
    shadowColor: '#000',
    shadowOffset: {width: 0, height: 2},
    shadowOpacity: 0.1,
    shadowRadius: 4,
    elevation: 3,
  },
  cardTitle: {
    fontSize: 18,
    fontWeight: '600',
    color: '#111827',
    marginBottom: 16,
  },
  detailRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 10,
    borderBottomWidth: 1,
    borderBottomColor: '#f3f4f6',
  },
  detailIcon: {
    width: 24,
    marginRight: 10,
  },
  detailLabel: {
    width: 90,
    fontSize: 14,
    color: '#6b7280',
    fontWeight: '500',
  },
  detailValue: {
    flex: 1,
    fontSize: 14,
    color: '#111827',
    textAlign: 'right',
  },
  monospace: {
    fontFamily: 'Courier',
    fontSize: 12,
    color: '#374151',
  },
  actionsCard: {
    backgroundColor: 'white',
    borderRadius: 12,
    margin: 16,
    padding: 16,
    shadowColor: '#000',
    shadowOffset: {width: 0, height: 2},
    shadowOpacity: 0.1,
    shadowRadius: 4,
    elevation: 3,
  },
  actionButton: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 12,
    borderBottomWidth: 1,
    borderBottomColor: '#f3f4f6',
  },
  deleteAction: {
    borderBottomWidth: 0,
  },
  actionIcon: {
    width: 44,
    height: 44,
    borderRadius: 22,
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 12,
  },
  actionInfo: {
    flex: 1,
  },
  actionLabel: {
    fontSize: 16,
    fontWeight: '500',
    color: '#111827',
    marginBottom: 2,
  },
  actionDescription: {
    fontSize: 13,
    color: '#6b7280',
  },
  bottomSpacer: {
    height: 32,
  },
});
