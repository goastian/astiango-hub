interface LComponentsAutoProbe {
  form: {
    name: string;
    url: string;
    query: string;
    queryPlaceholder: string;
    runOnCreate: string;
    viewport: string;
    viewports: {
      pc: {
        normal: string;
        wide: string;
        small: string;
      };
    };
    viewportWidth: string;
    viewportHeight: string;
    viewportPx: string;
  };
  task: {
    status: {
      label: {
        pending: string;
        running: string;
        completed: string;
        failed: string;
        cancelled: string;
        unknown: string;
      };
      tooltip: {
        pending: string;
        running: string;
        completed: string;
        failed: string;
        cancelled: string;
        unknown: string;
      };
    };
  };
  stats: {
    totalFields: string;
    totalLists: string;
    paginationType: string;
    noPagination: string;
  };
  navItems: {
    lists: string;
    fields: string;
    pagination: string;
    list: string;
    field: string;
  };
  patterns: {
    selectItem: string;
  };
  field: {
    title: string;
    name: string;
    selector: string;
    type: string;
    extractionType: string;
    attributeName: string;
    defaultValue: string;
    notFound: string;
    self: string;
  };
  list: {
    title: string;
    name: string;
    listSelector: string;
    listSelectorType: string;
    itemSelector: string;
    itemSelectorType: string;
    fields: string;
    nestedLists: string;
    notFound: string;
    self: string;
  };
  pagination: {
    title: string;
    type: string;
    selectorType: string;
    selector: string;
    maxPages: string;
    startPage: string;
    notFound: string;
  };
  pagePattern: {
    title: string;
    type: string;
    name: string;
    stats: string;
    fields: string;
    lists: string;
    hasPagination: string;
    notFound: string;
    fieldCount: string;
    types: {
      field: string;
      list: string;
      pagination: string;
    };
    selector: string;
    selectorType: string;
    selectorTypes: {
      css: string;
      xpath: string;
      regex: string;
    };
    extractionType: string;
    extractionTypes: {
      text: string;
      attribute: string;
      html: string;
    };
    attribute: string;
    elementType: string;
    elementTypes: {
      list: string;
      listItem: string;
      field: string;
      pagination: string;
    };
    displayConfig: {
      showLabel: string;
      focusMode: string;
    };
  };
}
