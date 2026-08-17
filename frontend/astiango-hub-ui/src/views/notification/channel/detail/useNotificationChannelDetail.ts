import { useDetail } from '@/layouts';

const useNotificationChannelDetail = () => {
  const ns: ListStoreNamespace = 'notificationChannel';

  return {
    ...useDetail<NotificationChannel>(ns),
  };
};

export default useNotificationChannelDetail;
