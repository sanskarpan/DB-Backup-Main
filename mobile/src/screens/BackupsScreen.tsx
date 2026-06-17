import React, {useEffect, useState} from 'react';
import {
  View,
  Text,
  StyleSheet,
  FlatList,
  TouchableOpacity,
  RefreshControl,
  ActivityIndicator,
  Alert,
} from 'react-native';
import Icon from 'react-native-vector-icons/Ionicons';
import {useDispatch, useSelector} from 'react-redux';
import {fetchBackups, createBackup} from '../store/backupsSlice';

export default function BackupsScreen({navigation}: any) {
  const [refreshing, setRefreshing] = useState(false);
  const dispatch = useDispatch();
  const {backups, loading, error} = useSelector((state: any) => state.backups);
  const {databases} = useSelector((state: any) => state.databases);

  useEffect(() => {
    dispatch(fetchBackups() as any);
  }, []);

  const onRefresh = async () => {
    setRefreshing(true);
    try {
      await dispatch(fetchBackups() as any);
    } finally {
      setRefreshing(false);
    }
  };

  const handleCreateBackup = () => {
    if (!databases || databases.length === 0) {
      Alert.alert(
        'No Databases',
        'Please add a database first before creating a backup.',
        [{text: 'OK'}],
      );
      return;
    }

    Alert.alert('Create Backup', 'Select a database to back up:', [
      ...databases.map((db: any) => ({
        text: db.name,
        onPress: () => {
          dispatch(createBackup({databaseId: db.id, options: {}}) as any);
        },
      })),
      {text: 'Cancel', style: 'cancel'},
    ]);
  };

  const renderItem = ({item}: {item: any}) => (
    <TouchableOpacity
      style={styles.backupCard}
      onPress={() => navigation.navigate('BackupDetail', {backup: item})}>
      <View style={styles.backupIcon}>
        <Icon name="folder" size={24} color="#3b82f6" />
      </View>
      <View style={styles.backupInfo}>
        <Text style={styles.backupName}>{item.database_name}</Text>
        <Text style={styles.backupTime}>
          {new Date(item.created_at).toLocaleString()}
        </Text>
        {item.size ? (
          <Text style={styles.backupSize}>
            {(item.size / 1024 / 1024).toFixed(2)} MB
          </Text>
        ) : null}
      </View>
      <StatusBadge status={item.status} />
    </TouchableOpacity>
  );

  const renderEmpty = () => {
    if (loading) {
      return (
        <View style={styles.emptyState}>
          <ActivityIndicator size="large" color="#3b82f6" />
        </View>
      );
    }
    return (
      <View style={styles.emptyState}>
        <Icon name="folder-open-outline" size={64} color="#9ca3af" />
        <Text style={styles.emptyText}>No backups yet</Text>
        <Text style={styles.emptySubText}>
          Pull down to refresh or tap + to create a backup
        </Text>
      </View>
    );
  };

  return (
    <View style={styles.container}>
      <View style={styles.header}>
        <Text style={styles.headerTitle}>Backups</Text>
        <Text style={styles.headerSubtitle}>
          {backups.length} backup{backups.length !== 1 ? 's' : ''}
        </Text>
      </View>

      {error ? (
        <View style={styles.errorBanner}>
          <Icon name="warning-outline" size={16} color="#991b1b" />
          <Text style={styles.errorText}>{error}</Text>
        </View>
      ) : null}

      <FlatList
        data={backups}
        keyExtractor={item => item.id}
        renderItem={renderItem}
        ListEmptyComponent={renderEmpty}
        contentContainerStyle={
          backups.length === 0 ? styles.emptyContainer : styles.listContent
        }
        refreshControl={
          <RefreshControl refreshing={refreshing} onRefresh={onRefresh} />
        }
      />

      <TouchableOpacity style={styles.fab} onPress={handleCreateBackup}>
        <Icon name="add" size={28} color="white" />
      </TouchableOpacity>
    </View>
  );
}

function StatusBadge({status}: {status: string}) {
  const colors: Record<string, {bg: string; text: string}> = {
    completed: {bg: '#d1fae5', text: '#065f46'},
    running: {bg: '#dbeafe', text: '#1e40af'},
    failed: {bg: '#fee2e2', text: '#991b1b'},
    pending: {bg: '#fef3c7', text: '#92400e'},
  };

  const color = colors[status] || colors.pending;

  return (
    <View style={[styles.badge, {backgroundColor: color.bg}]}>
      <Text style={[styles.badgeText, {color: color.text}]}>
        {status.charAt(0).toUpperCase() + status.slice(1)}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f9fafb',
  },
  header: {
    backgroundColor: '#3b82f6',
    padding: 24,
    paddingTop: 48,
  },
  headerTitle: {
    fontSize: 28,
    fontWeight: 'bold',
    color: 'white',
  },
  headerSubtitle: {
    fontSize: 16,
    color: '#dbeafe',
    marginTop: 4,
  },
  errorBanner: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: '#fee2e2',
    padding: 12,
    margin: 16,
    borderRadius: 8,
    gap: 8,
  },
  errorText: {
    flex: 1,
    fontSize: 14,
    color: '#991b1b',
  },
  listContent: {
    padding: 16,
  },
  emptyContainer: {
    flex: 1,
  },
  backupCard: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: 'white',
    borderRadius: 12,
    padding: 16,
    marginBottom: 12,
    shadowColor: '#000',
    shadowOffset: {width: 0, height: 2},
    shadowOpacity: 0.1,
    shadowRadius: 4,
    elevation: 3,
  },
  backupIcon: {
    width: 48,
    height: 48,
    borderRadius: 24,
    backgroundColor: '#dbeafe',
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 12,
  },
  backupInfo: {
    flex: 1,
  },
  backupName: {
    fontSize: 16,
    fontWeight: '600',
    color: '#111827',
    marginBottom: 4,
  },
  backupTime: {
    fontSize: 13,
    color: '#6b7280',
  },
  backupSize: {
    fontSize: 12,
    color: '#9ca3af',
    marginTop: 2,
  },
  badge: {
    paddingHorizontal: 12,
    paddingVertical: 6,
    borderRadius: 12,
  },
  badgeText: {
    fontSize: 12,
    fontWeight: '600',
  },
  emptyState: {
    alignItems: 'center',
    padding: 48,
  },
  emptyText: {
    marginTop: 16,
    fontSize: 18,
    fontWeight: '600',
    color: '#374151',
  },
  emptySubText: {
    marginTop: 8,
    fontSize: 14,
    color: '#9ca3af',
    textAlign: 'center',
  },
  fab: {
    position: 'absolute',
    bottom: 24,
    right: 24,
    width: 56,
    height: 56,
    borderRadius: 28,
    backgroundColor: '#3b82f6',
    justifyContent: 'center',
    alignItems: 'center',
    shadowColor: '#000',
    shadowOffset: {width: 0, height: 4},
    shadowOpacity: 0.3,
    shadowRadius: 6,
    elevation: 6,
  },
});
