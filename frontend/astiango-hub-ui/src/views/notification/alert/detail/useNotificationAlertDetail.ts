import { useDetail } from '@/layouts';

const useNotificationAlertDetail = () => {
  const ns: ListStoreNamespace = 'notificationAlert';

  return {
    ...useDetail<NotificationAlert>(ns),
  };
};

export default useNotificationAlertDetail;
