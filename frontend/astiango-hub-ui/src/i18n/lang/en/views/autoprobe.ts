const autoprobe: LViewsAutoProbe = {
  table: {
    columns: {
      name: 'Name',
      url: 'URL',
      query: 'Query',
      status: 'Status',
      lastTask: 'Last Task',
      patterns: 'Patterns',
    },
  },
  navActions: {
    new: {
      label: 'New AutoProbe',
      tooltip: 'Create a new AutoProbe',
    },
    filter: {
      search: {
        placeholder: 'Search by name',
      },
    },
    run: {
      label: 'Run AutoProbe',
      tooltip: 'Run the selected AutoProbe',
    },
  },
};

export default autoprobe;
