const database: LComponentsDatabase = {
  label: {
    text: 'Database',
    tooltip: 'Database',
  },
  form: {
    name: 'Name',
    description: 'Description',
    dataSource: 'Database Type',
    status: 'Status',
    host: 'Host',
    port: 'Port',
    url: 'URL',
    hosts: 'Hosts',
    username: 'Username',
    password: 'Password',
    address: 'Address',
    changePassword: 'Change Password',
    database: 'Database Name',
    useOrm: 'Use ORM',
    ormMode: 'Database Engine',
    ormModeTooltip: 'Use modern ORM for better type safety and performance',
    mongo: {
      authSource: 'Auth Source',
      authMechanism: 'Auth Mechanism',
    },
    mysql: {
      charset: 'Charset',
      parseTime: 'Parse Time',
    },
    postgresql: {
      sslMode: 'SSL Mode',
    },
    default: {
      host: 'Default Host',
      port: 'Default Port',
      url: 'Default URL',
      database: 'Default Database',
    },
  },
  dataSources: {
    default: 'Default',
    mongo: 'MongoDB',
    mysql: 'MySQL',
    postgres: 'PostgreSQL',
    mssql: 'Microsoft SQL Server',
    elasticsearch: 'ElasticSearch',
    kafka: 'Kafka',
    redis: 'Redis',
  },
  status: {
    label: {
      online: 'Online',
      offline: 'Offline',
      unknown: 'Unknown',
    },
    tooltip: {
      online: 'Data source is currently online',
      offline: 'Data source is currently offline',
      unknown: 'Unknown data source status',
    },
  },
  default: {
    name: 'Default Database',
    host: 'Default Host',
    port: 'Default Port',
  },
  message: {
    success: {
      change: 'Changed data source successfully',
    },
    warning: {
      emptyQuery: 'Query is empty. Please select a query to run',
    },
  },
  messageBox: {
    confirm: {
      renameTable: {
        title: 'Rename Table',
        message: 'Are you sure you want to rename the table?',
      },
    },
    prompt: {
      dropTable: {
        title: 'Drop Table',
        message: 'Are you sure you want to drop the table "{tableName}"?',
        placeholder: 'Please type the table name "{tableName}" to confirm',
        error: 'Table name does not match',
      },
    },
  },
  connectType: {
    label: {
      standard: 'Standard',
      url: 'URL',
      hosts: 'Hosts',
    },
    tips: {
      standard:
        'Standard connect settings, normally used for single instance configurations',
      url: 'Connect settings with URL, suitable for more complex connect settings',
      hosts:
        'Hosts connect settings, normally used for multiple instances or cluster configurations',
    },
  },
  databases: {
    database: {
      name: 'Database Name',
      tables: {
        name: 'Table Name',
        columns: 'Columns',
        indexes: 'Indexes',
      },
      create: {
        name: 'New database name',
      },
    },
    table: {
      name: 'Table Name',
      columns: {
        name: 'Name',
        type: 'Data Type',
        notNull: 'Not Null',
        default: 'Default Expression',
        primary: 'PK',
        autoIncrement: 'Auto Inc',
      },
      indexes: {
        name: 'Name',
        type: 'Type',
        columns: 'Columns',
        unique: 'Unique',
        column: {
          name: 'Column Name',
          order: 'Column Order',
        },
      },
      create: {
        name: 'New table name',
      },
      actions: {
        addColumn: 'Add Column',
        editColumns: 'Edit Columns',
        editIndexColumns: 'Edit Index Columns',
        editIndexes: 'Edit Indexes',
        truncate: 'Truncate',
        drop: 'Drop',
      },
    },
    labels: {
      columns: 'Columns',
      indexes: 'Indexes',
    },
  },
  actions: {
    commitChanges: 'Commit Changes',
    rollbackChanges: 'Rollback Changes',
    runQuery: 'Run Query',
  },
  orm: {
    enabled: 'ORM (Recommended)',
    disabled: 'Legacy SQL',
    modern: 'Type-safe, optimized',
    legacy: 'Traditional',
    helpTextEnabled: 'Using modern ORM with type safety and performance optimization',
    helpTextDisabled: 'Using legacy SQL queries (consider upgrading to ORM)',
    switchToOrm: 'Switch to ORM',
    switchToLegacy: 'Switch to Legacy',
    benefitsTitle: 'ORM Benefits Active',
    benefit1: 'Type-safe database operations',
    benefit2: '60% better memory efficiency',
    benefit3: 'Automatic SQL injection prevention',
    benefit4: 'Advanced error detection',
    migrationTitle: 'Safe Migration',
    migrationMessage: 'You can switch between ORM and Legacy modes anytime without data loss. ORM provides better performance and safety.',
    notSupportedTitle: 'ORM Not Available',
    notSupportedMessage: 'The ORM (Object-Relational Mapping) system is only available for relational databases. This database type uses the traditional database service instead.',
  },
};

export default database;
