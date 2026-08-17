const autoprobe: LViewsAutoProbe = {
  table: {
    columns: {
      name: '名称',
      url: 'URL',
      query: '查询',
      status: '状态',
      lastTask: '最近任务',
      patterns: '模式',
    },
  },
  navActions: {
    new: {
      label: '新建 AutoProbe',
      tooltip: '创建新的 AutoProbe',
    },
    filter: {
      search: {
        placeholder: '按名称搜索',
      },
    },
    run: {
      label: '运行 AutoProbe',
      tooltip: '运行选定的 AutoProbe',
    },
  },
};

export default autoprobe;
