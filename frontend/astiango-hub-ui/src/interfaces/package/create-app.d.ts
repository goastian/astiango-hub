import { Store } from 'vuex';

export declare global {
  interface CreateAppOptions {
    mount?: boolean | string;
    store?: Store;
    rootRoutes?: Array<ExtendedRouterRecord>;
    routes?: Array<ExtendedRouterRecord>;
    allRoutes?: Array<ExtendedRouterRecord>;
    createRouterOptions?: CreateRouterOptions;
  }
}
