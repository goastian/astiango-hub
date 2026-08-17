type NotificationAlertStoreModule = BaseModule<
  NotificationAlertStoreState,
  NotificationAlertStoreGetters,
  NotificationAlertStoreMutations,
  NotificationAlertStoreActions
>;

interface NotificationAlertStoreState
  extends BaseStoreState<NotificationAlert> {
  allAlerts: NotificationAlert[];
}

type NotificationAlertStoreGetters = BaseStoreGetters<NotificationAlert>;

interface NotificationAlertStoreMutations
  extends BaseStoreMutations<NotificationAlert> {
  setAllAlerts: StoreMutation<
    NotificationSettingStoreState,
    NotificationAlert[]
  >;
  resetAllAlerts: StoreMutation<NotificationAlertStoreState>;
}

interface NotificationAlertStoreActions
  extends BaseStoreActions<NotificationAlert> {
  getAllAlerts: StoreAction<NotificationAlertStoreState>;
}
