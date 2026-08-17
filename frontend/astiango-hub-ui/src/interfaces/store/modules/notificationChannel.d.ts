type NotificationChannelStoreModule = BaseModule<
  NotificationChannelStoreState,
  NotificationChannelStoreGetters,
  NotificationChannelStoreMutations,
  NotificationChannelStoreActions
>;

interface NotificationChannelStoreState
  extends BaseStoreState<NotificationChannel> {
  allChannels: NotificationChannel[];
}

type NotificationChannelStoreGetters = BaseStoreGetters<NotificationChannel>;

interface NotificationChannelStoreMutations
  extends BaseStoreMutations<NotificationChannel> {
  setAllChannels: StoreMutation<
    NotificationChannelStoreState,
    NotificationChannel[]
  >;
  resetAllChannels: StoreMutation<NotificationChannelStoreState>;
}

interface NotificationChannelStoreActions
  extends BaseStoreActions<NotificationChannel> {
  sendTestMessage: StoreAction<
    NotificationChannelStoreState,
    { id: string; toMail?: string }
  >;
  getAllChannels: StoreAction<NotificationChannelStoreState>;
}
