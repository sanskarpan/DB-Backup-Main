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
  Modal,
  TextInput,
  KeyboardAvoidingView,
  Platform,
} from 'react-native';
import Icon from 'react-native-vector-icons/Ionicons';
import {useDispatch, useSelector} from 'react-redux';
import {
  fetchDatabases,
  deleteDatabase,
  createDatabase,
} from '../store/databasesSlice';

const DB_TYPES = [
  'postgres',
  'mysql',
  'mongodb',
  'redis',
  'sqlite',
];

const emptyForm = {
  name: '',
  type: 'postgres',
  host: '',
  port: '',
  username: '',
  database_name: '',
};

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
  const [modalVisible, setModalVisible] = useState(false);
  const [form, setForm] = useState(emptyForm);
  const [submitting, setSubmitting] = useState(false);
  const dispatch = useDispatch();
  const {databases, loading, error} = useSelector(
    (state: any) => state.databases,
  );

  useEffect(() => {
    dispatch(fetchDatabases() as any);
  }, []);

  const openAddModal = () => {
    setForm(emptyForm);
    setModalVisible(true);
  };

  const updateField = (field: keyof typeof emptyForm, value: string) =>
    setForm(prev => ({...prev, [field]: value}));

  const handleCreateDatabase = async () => {
    if (!form.name.trim() || !form.type.trim() || !form.host.trim()) {
      Alert.alert('Missing fields', 'Name, type and host are required.');
      return;
    }

    const payload: any = {
      name: form.name.trim(),
      type: form.type.trim().toLowerCase(),
      host: form.host.trim(),
    };
    if (form.port.trim()) {
      const parsedPort = parseInt(form.port.trim(), 10);
      if (Number.isNaN(parsedPort)) {
        Alert.alert('Invalid port', 'Port must be a number.');
        return;
      }
      payload.port = parsedPort;
    }
    if (form.username.trim()) {
      payload.username = form.username.trim();
    }
    if (form.database_name.trim()) {
      payload.database_name = form.database_name.trim();
    }

    setSubmitting(true);
    try {
      const result: any = await dispatch(createDatabase(payload) as any);
      if (result?.error) {
        Alert.alert(
          'Failed',
          result.payload || 'Could not create the database connection.',
        );
        return;
      }
      setModalVisible(false);
      setForm(emptyForm);
      // Refresh the list from the backend.
      dispatch(fetchDatabases() as any);
    } finally {
      setSubmitting(false);
    }
  };

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

      <TouchableOpacity style={styles.fab} onPress={openAddModal}>
        <Icon name="add" size={28} color="white" />
      </TouchableOpacity>

      <Modal
        visible={modalVisible}
        animationType="slide"
        transparent
        onRequestClose={() => setModalVisible(false)}>
        <KeyboardAvoidingView
          style={styles.modalOverlay}
          behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
          <View style={styles.modalCard}>
            <View style={styles.modalHeader}>
              <Text style={styles.modalTitle}>Add Database</Text>
              <TouchableOpacity onPress={() => setModalVisible(false)}>
                <Icon name="close" size={24} color="#6b7280" />
              </TouchableOpacity>
            </View>

            <Text style={styles.fieldLabel}>Name *</Text>
            <TextInput
              style={styles.input}
              value={form.name}
              onChangeText={v => updateField('name', v)}
              placeholder="Production DB"
              placeholderTextColor="#9ca3af"
              autoCapitalize="none"
            />

            <Text style={styles.fieldLabel}>Type *</Text>
            <View style={styles.typeRow}>
              {DB_TYPES.map(t => (
                <TouchableOpacity
                  key={t}
                  style={[
                    styles.typeChip,
                    form.type === t && styles.typeChipActive,
                  ]}
                  onPress={() => updateField('type', t)}>
                  <Text
                    style={[
                      styles.typeChipText,
                      form.type === t && styles.typeChipTextActive,
                    ]}>
                    {t}
                  </Text>
                </TouchableOpacity>
              ))}
            </View>

            <Text style={styles.fieldLabel}>Host *</Text>
            <TextInput
              style={styles.input}
              value={form.host}
              onChangeText={v => updateField('host', v)}
              placeholder="localhost"
              placeholderTextColor="#9ca3af"
              autoCapitalize="none"
              autoCorrect={false}
            />

            <Text style={styles.fieldLabel}>Port</Text>
            <TextInput
              style={styles.input}
              value={form.port}
              onChangeText={v => updateField('port', v)}
              placeholder="5432"
              placeholderTextColor="#9ca3af"
              keyboardType="number-pad"
            />

            <Text style={styles.fieldLabel}>Username</Text>
            <TextInput
              style={styles.input}
              value={form.username}
              onChangeText={v => updateField('username', v)}
              placeholder="postgres"
              placeholderTextColor="#9ca3af"
              autoCapitalize="none"
              autoCorrect={false}
            />

            <Text style={styles.fieldLabel}>Database name</Text>
            <TextInput
              style={styles.input}
              value={form.database_name}
              onChangeText={v => updateField('database_name', v)}
              placeholder="app_production"
              placeholderTextColor="#9ca3af"
              autoCapitalize="none"
              autoCorrect={false}
            />

            <TouchableOpacity
              style={[
                styles.submitButton,
                submitting && styles.submitButtonDisabled,
              ]}
              onPress={handleCreateDatabase}
              disabled={submitting}>
              {submitting ? (
                <ActivityIndicator color="white" />
              ) : (
                <Text style={styles.submitButtonText}>Add Database</Text>
              )}
            </TouchableOpacity>
          </View>
        </KeyboardAvoidingView>
      </Modal>
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
  modalOverlay: {
    flex: 1,
    backgroundColor: 'rgba(0,0,0,0.4)',
    justifyContent: 'flex-end',
  },
  modalCard: {
    backgroundColor: 'white',
    borderTopLeftRadius: 20,
    borderTopRightRadius: 20,
    padding: 20,
    paddingBottom: 32,
  } as any,
  modalHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 12,
  },
  modalTitle: {
    fontSize: 20,
    fontWeight: '700',
    color: '#111827',
  },
  fieldLabel: {
    fontSize: 13,
    fontWeight: '600',
    color: '#374151',
    marginTop: 12,
    marginBottom: 6,
  },
  input: {
    height: 44,
    borderWidth: 1,
    borderColor: '#d1d5db',
    borderRadius: 8,
    paddingHorizontal: 12,
    fontSize: 14,
    color: '#111827',
    backgroundColor: '#f9fafb',
  },
  typeRow: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 8,
  },
  typeChip: {
    paddingHorizontal: 12,
    paddingVertical: 8,
    borderRadius: 16,
    borderWidth: 1,
    borderColor: '#d1d5db',
    backgroundColor: '#f9fafb',
  },
  typeChipActive: {
    backgroundColor: '#3b82f6',
    borderColor: '#3b82f6',
  },
  typeChipText: {
    fontSize: 13,
    color: '#374151',
  },
  typeChipTextActive: {
    color: 'white',
    fontWeight: '600',
  },
  submitButton: {
    marginTop: 20,
    height: 48,
    backgroundColor: '#3b82f6',
    borderRadius: 10,
    justifyContent: 'center',
    alignItems: 'center',
  },
  submitButtonDisabled: {
    opacity: 0.6,
  },
  submitButtonText: {
    color: 'white',
    fontWeight: '600',
    fontSize: 16,
  },
});
