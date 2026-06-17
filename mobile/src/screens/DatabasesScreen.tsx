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
import {fetchDatabases, deleteDatabase} from '../store/databasesSlice';

const DB_TYPE_ICONS: Record<string, string> = {
  postgres: 'server',
  postgresql: 'server',
  mysql: 'server',
  mongodb: 'server',
  redis: 'server',
  sqlite: 'server',
};

const DB_TYPE_COLORS: Record<string, string> = {
  postgres: '#336791',
  postgresql: '#336791',
  mysql: '#4479a1',
  mongodb: '#47a248',
  redis: '#dc382d',
  sqlite: '#003b57',
};

export default function DatabasesScreen({navigation}: any) {
  const [refreshing, setRefreshing] = useState(false);
  const dispatch = useDispatch();
  const {databases, loading, error} = useSelector(
    (state: any) => state.databases,
  );

  useEffect(() => {
    dispatch(fetchDatabases() as any);
  }, []);

  const onRefresh = async () => {
    setRefreshing(true);
    try {
      await dispatch(fetchDatabases() as any);
    } finally {
      setRefreshing(false);
    }
  };

  const handleDeleteDatabase = (db: any) => {
    Alert.alert(
      'Delete Database',
      `Are you sure you want to remove "${db.name}"? This will not delete the actual database, only this connection.`,
      [
        {text: 'Cancel', style: 'cancel'},
        {
          text: 'Delete',
          style: 'destructive',
          onPress: () => dispatch(deleteDatabase(db.id) as any),
        },
      ],
    );
  };

  const getDbColor = (type: string) =>
    DB_TYPE_COLORS[type?.toLowerCase()] || '#6b7280';

  const renderItem = ({item}: {item: any}) => (
    <View style={styles.dbCard}>
      <View
        style={[
          styles.dbIcon,
          {backgroundColor: `${getDbColor(item.type)}20`},
        ]}>
        <Icon
          name={DB_TYPE_ICONS[item.type?.toLowerCase()] || 'server'}
          size={24}
          color={getDbColor(item.type)}
        />
      </View>
      <View style={styles.dbInfo}>
        <Text style={styles.dbName}>{item.name}</Text>
        <Text style={styles.dbType}>
          {item.type?.toUpperCase() || 'UNKNOWN'}
        </Text>
        <Text style={styles.dbHost}>
          {item.host}
          {item.port ? `:${item.port}` : ''}
          {item.database_name ? ` / ${item.database_name}` : ''}
        </Text>
      </View>
      <View style={styles.dbActions}>
        {item.status ? (
          <StatusDot status={item.status} />
        ) : null}
        <TouchableOpacity
          style={styles.deleteButton}
          onPress={() => handleDeleteDatabase(item)}>
          <Icon name="trash-outline" size={18} color="#ef4444" />
        </TouchableOpacity>
      </View>
    </View>
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
        <Icon name="server-outline" size={64} color="#9ca3af" />
        <Text style={styles.emptyText}>No databases connected</Text>
        <Text style={styles.emptySubText}>
          Pull down to refresh or tap + to add a database
        </Text>
      </View>
    );
  };

  return (
    <View style={styles.container}>
      <View style={styles.header}>
        <Text style={styles.headerTitle}>Databases</Text>
        <Text style={styles.headerSubtitle}>
          {databases.length} connection{databases.length !== 1 ? 's' : ''}
        </Text>
      </View>

      {error ? (
        <View style={styles.errorBanner}>
          <Icon name="warning-outline" size={16} color="#991b1b" />
          <Text style={styles.errorText}>{error}</Text>
        </View>
      ) : null}

      <FlatList
        data={databases}
        keyExtractor={item => item.id}
        renderItem={renderItem}
        ListEmptyComponent={renderEmpty}
        contentContainerStyle={
          databases.length === 0 ? styles.emptyContainer : styles.listContent
        }
        refreshControl={
          <RefreshControl refreshing={refreshing} onRefresh={onRefresh} />
        }
      />

      <TouchableOpacity
        style={styles.fab}
        onPress={() =>
          Alert.alert('Add Database', 'Feature coming soon.', [{text: 'OK'}])
        }>
        <Icon name="add" size={28} color="white" />
      </TouchableOpacity>
    </View>
  );
}

function StatusDot({status}: {status: string}) {
  const color =
    status === 'connected'
      ? '#10b981'
      : status === 'error'
      ? '#ef4444'
      : '#f59e0b';

  return <View style={[styles.statusDot, {backgroundColor: color}]} />;
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
  dbCard: {
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
  dbIcon: {
    width: 48,
    height: 48,
    borderRadius: 24,
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 12,
  },
  dbInfo: {
    flex: 1,
  },
  dbName: {
    fontSize: 16,
    fontWeight: '600',
    color: '#111827',
    marginBottom: 2,
  },
  dbType: {
    fontSize: 12,
    fontWeight: '700',
    color: '#6b7280',
    marginBottom: 2,
    letterSpacing: 0.5,
  },
  dbHost: {
    fontSize: 13,
    color: '#9ca3af',
  },
  dbActions: {
    alignItems: 'center',
    gap: 8,
  },
  statusDot: {
    width: 10,
    height: 10,
    borderRadius: 5,
  },
  deleteButton: {
    padding: 4,
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
