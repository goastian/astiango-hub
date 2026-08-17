type AutoProbeStoreModule = BaseModule<
  AutoProbeStoreState,
  AutoProbeStoreGetters,
  AutoProbeStoreMutations,
  AutoProbeStoreActions
>;

interface AutoProbeStoreState extends BaseStoreState<AutoProbeV2> {
  pagePattern?: PagePatternV2;
  pagePatternData?: PatternDataV2[];
}

type AutoProbeStoreGetters = BaseStoreGetters<AutoProbeV2>;

interface AutoProbeStoreMutations extends BaseStoreMutations<AutoProbeV2> {
  setPagePattern: StoreMutation<AutoProbeStoreState, PagePatternV2>;
  resetPagePattern: StoreMutation<AutoProbeStoreState>;
  setPagePatternData: StoreMutation<AutoProbeStoreState, PatternDataV2[]>;
  resetPagePatternData: StoreMutation<AutoProbeStoreState>;
}

interface AutoProbeStoreActions extends BaseStoreActions<AutoProbeV2> {
  runTask: StoreAction<AutoProbeStoreState, { id: string }>;
  cancelTask: StoreAction<AutoProbeStoreState, { id: string }>;
  getPagePattern: StoreAction<AutoProbeStoreState, { id: string }>;
  getPagePatternData: StoreAction<AutoProbeStoreState, { id: string }>;
}
