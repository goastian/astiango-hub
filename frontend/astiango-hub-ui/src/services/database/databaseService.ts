import { Store } from 'vuex';
import { getDefaultService } from '@/utils';
import useRequest from '@/services/request';

const useDataSourceService = (
  store: Store<RootStoreState>
): Services<Database> => {
  const ns: ListStoreNamespace = 'database';

  return {
    ...getDefaultService<Database>(ns, store),
  };
};

export const useDatabaseOrmService = () => {
  const { get, put, post } = useRequest();

  // Check ORM compatibility for a database
  const getOrmCompatibility = async (databaseId: string): Promise<DatabaseOrmCompatibility> => {
    const res = await get(`/databases/${databaseId}/orm/compatibility`);
    return res.data;
  };

  // Get current ORM status for a database
  const getOrmStatus = async (databaseId: string): Promise<DatabaseOrmStatus> => {
    const res = await get(`/databases/${databaseId}/orm/status`);
    return res.data;
  };

  // Toggle ORM on/off for a database
  const setOrmStatus = async (databaseId: string, enabled: boolean): Promise<void> => {
    await put(`/databases/${databaseId}/orm/status`, { enabled });
  };

  // Initialize ORM settings with intelligent defaults
  const initializeOrm = async (databaseId: string): Promise<void> => {
    await post(`/databases/${databaseId}/orm/initialize`);
  };

  // Check if data source supports ORM (client-side helper)
  const isOrmSupported = (dataSource?: string): boolean => {
    if (!dataSource) return false;
    return ['mysql', 'postgres', 'mssql'].includes(dataSource.toLowerCase());
  };

  return {
    getOrmCompatibility,
    getOrmStatus,
    setOrmStatus,
    initializeOrm,
    isOrmSupported,
  };
};

export default useDataSourceService;
