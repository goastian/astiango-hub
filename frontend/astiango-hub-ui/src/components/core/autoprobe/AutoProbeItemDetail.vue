<script setup lang="tsx">
import { computed } from 'vue';
import { ClNavLink, ClIcon, ClAutoProbeSelector } from '@/components';
import { translate } from '@/utils';
import { CellStyle, ColumnStyle } from 'element-plus';

const props = defineProps<{
  item: AutoProbeNavItem;
  activeId?: string;
}>();

const emit = defineEmits<{
  (e: 'row-click', id: string): void;
}>();

const t = translate;

const tableColumns = computed<TableColumns<AutoProbeNavItem>>(() => {
  return [
    {
      key: 'name',
      label: t('components.autoprobe.pagePattern.name'),
      width: '200',
      value: (row: AutoProbeNavItem) => {
        let icon: Icon;
        switch (row.type) {
          case 'field':
            icon = ['fa', 'tag'];
            break;
          case 'list':
            icon = ['fa', 'list'];
            break;
          case 'pagination':
            icon = ['fa', 'ellipsis-h'];
            break;
          default:
            icon = ['fa', 'question'];
        }
        return (
          <ClNavLink
            onClick={() => {
              emit('row-click', row.id);
            }}
          >
            <span style={{ marginRight: '5px' }}>
              <ClIcon icon={icon} />
            </span>
            {row.label}
          </ClNavLink>
        );
      },
    },
    {
      key: 'type',
      label: t('components.autoprobe.pagePattern.type'),
      width: '100',
      value: (row: AutoProbeNavItem) => {
        return t(`components.autoprobe.pagePattern.types.${row.type}`);
      },
    },
    {
      key: 'fields',
      label: t('components.autoprobe.pagePattern.fieldCount'),
      width: '80',
      value: (row: AutoProbeNavItem) => {
        if (row.type === 'list') {
          return (
            <ClNavLink
              label={row.fieldCount}
              onClick={() => {
                emit('row-click', row.id);
              }}
            />
          );
        }
      },
    },
    {
      key: 'selector',
      label: t('components.autoprobe.pagePattern.selector'),
      width: 'auto',
      minWidth: '300',
      value: (row: AutoProbeNavItem) => {
        switch (row.type) {
          case 'list':
            const list = row.rule as ListRule;
            const selectorType = list.item_selector_type;
            const selector = [list.list_selector, list.item_selector]
              .filter(item => item)
              .join(' > ');
            return (
              <ClAutoProbeSelector
                selectorType={selectorType}
                selector={selector}
                onClick={() => {
                  emit('row-click', row.id);
                }}
              />
            );
          case 'field':
            const field = row.rule as FieldRule;
            return (
              <ClAutoProbeSelector
                selectorType={field.selector_type}
                selector={field.selector}
                extractType={field.extraction_type}
                attribute={field.attribute_name}
                onClick={() => {
                  emit('row-click', row.id);
                }}
              />
            );
          case 'pagination':
            const pagination = row.rule as PaginationRule;
            return (
              <ClAutoProbeSelector
                selectorType={pagination.selector_type}
                selector={pagination.selector}
                onClick={() => {
                  emit('row-click', row.id);
                }}
              />
            );
        }
      },
    },
  ] as TableColumns<AutoProbeNavItem>;
});
const tableData = computed<TableData<AutoProbeNavItem>>(() => {
  const { item } = props;
  if (!item.children) return [];
  return item?.children.map((item: AutoProbeNavItem) => {
    const row = {
      id: item.id,
      name: item.name,
      label: item.name,
      type: item.type,
      rule: item.rule,
    } as AutoProbeNavItem;
    if (item.type === 'list') {
      row.fieldCount = item.children?.length || 0;
    }
    return row;
  }) as TableData<AutoProbeNavItem>;
});

const rowStyle: ColumnStyle<AutoProbeNavItem> = ({ row }) => {
  const { activeId } = props;
  if (row.id === activeId) {
    return {
      backgroundColor: 'var(--el-color-primary-light-9)',
    };
  }
  return {};
};

const cellStyle: CellStyle<AutoProbeNavItem> = ({ row }) => {
  const { activeId } = props;
  if (row.id === activeId) {
    return {
      backgroundColor: 'var(--el-color-primary-light-9)',
    };
  }
  return {};
};

defineOptions({ name: 'ClAutoProbeItemDetail' });
</script>

<template>
  <div class="autoprobe-page-pattern-detail">
    <cl-table
      :columns="tableColumns"
      :data="tableData"
      embedded
      sticky-header
      hide-footer
      :row-style="rowStyle"
      :cell-style="cellStyle"
    />
  </div>
</template>

<style scoped>
.autoprobe-page-pattern-detail {
  height: 100%;
}
</style>
