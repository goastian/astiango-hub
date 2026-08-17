import { useDetail } from '@/layouts';

const useNotificationSettingDetail = () => {
  const ns: ListStoreNamespace = 'notificationSetting';

  return {
    ...useDetail<NotificationSetting>(ns),
  };
};

export default useNotificationSettingDetail;
