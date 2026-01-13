import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Table, Pagination, Column } from './Table';

interface TestData {
  id: string;
  name: string;
  status: string;
  count: number;
}

const mockData: TestData[] = [
  { id: '1', name: 'Item 1', status: 'Active', count: 10 },
  { id: '2', name: 'Item 2', status: 'Pending', count: 25 },
  { id: '3', name: 'Item 3', status: 'Completed', count: 5 },
];

const mockColumns: Column<TestData>[] = [
  {
    key: 'name',
    header: 'Name',
    accessor: (item) => item.name,
    sortable: true,
  },
  {
    key: 'status',
    header: 'Status',
    accessor: (item) => item.status,
    sortable: true,
  },
  {
    key: 'count',
    header: 'Count',
    accessor: (item) => item.count,
    sortable: false,
    align: 'right',
  },
];

describe('Table', () => {
  describe('Rendering', () => {
    it('renders table with data', () => {
      render(
        <Table
          columns={mockColumns}
          data={mockData}
          keyExtractor={(item) => item.id}
        />
      );

      expect(screen.getByText('Item 1')).toBeInTheDocument();
      expect(screen.getByText('Item 2')).toBeInTheDocument();
      expect(screen.getByText('Item 3')).toBeInTheDocument();
    });

    it('renders column headers', () => {
      render(
        <Table
          columns={mockColumns}
          data={mockData}
          keyExtractor={(item) => item.id}
        />
      );

      expect(screen.getByText('Name')).toBeInTheDocument();
      expect(screen.getByText('Status')).toBeInTheDocument();
      expect(screen.getByText('Count')).toBeInTheDocument();
    });

    it('renders cells with accessor values', () => {
      render(
        <Table
          columns={mockColumns}
          data={mockData}
          keyExtractor={(item) => item.id}
        />
      );

      expect(screen.getByText('Active')).toBeInTheDocument();
      expect(screen.getByText('Pending')).toBeInTheDocument();
      expect(screen.getByText('Completed')).toBeInTheDocument();
      expect(screen.getByText('10')).toBeInTheDocument();
      expect(screen.getByText('25')).toBeInTheDocument();
      expect(screen.getByText('5')).toBeInTheDocument();
    });

    it('uses keyExtractor for row keys', () => {
      const { container } = render(
        <Table
          columns={mockColumns}
          data={mockData}
          keyExtractor={(item) => item.id}
        />
      );
      const rows = container.querySelectorAll('tbody tr');

      expect(rows.length).toBe(3);
    });

    it('renders with custom className', () => {
      const { container } = render(
        <Table
          columns={mockColumns}
          data={mockData}
          keyExtractor={(item) => item.id}
          className="custom-table"
        />
      );
      const wrapper = container.firstChild as HTMLElement;

      expect(wrapper).toHaveClass('custom-table');
    });
  });

  describe('Column alignment', () => {
    it('renders left-aligned column by default', () => {
      const columns: Column<TestData>[] = [
        {
          key: 'name',
          header: 'Name',
          accessor: (item) => item.name,
        },
      ];

      const { container } = render(
        <Table
          columns={columns}
          data={mockData}
          keyExtractor={(item) => item.id}
        />
      );
      const headerCell = container.querySelector('th');
      const dataCell = container.querySelector('td');

      expect(headerCell).toHaveClass('text-left');
      expect(dataCell).toHaveClass('text-left');
    });

    it('renders center-aligned column', () => {
      const columns: Column<TestData>[] = [
        {
          key: 'name',
          header: 'Name',
          accessor: (item) => item.name,
          align: 'center',
        },
      ];

      const { container } = render(
        <Table
          columns={columns}
          data={mockData}
          keyExtractor={(item) => item.id}
        />
      );
      const headerCell = container.querySelector('th');
      const dataCell = container.querySelector('td');

      expect(headerCell).toHaveClass('text-center');
      expect(dataCell).toHaveClass('text-center');
    });

    it('renders right-aligned column', () => {
      const columns: Column<TestData>[] = [
        {
          key: 'name',
          header: 'Name',
          accessor: (item) => item.name,
          align: 'right',
        },
      ];

      const { container } = render(
        <Table
          columns={columns}
          data={mockData}
          keyExtractor={(item) => item.id}
        />
      );
      const headerCell = container.querySelector('th');
      const dataCell = container.querySelector('td');

      expect(headerCell).toHaveClass('text-right');
      expect(dataCell).toHaveClass('text-right');
    });
  });

  describe('Column width', () => {
    it('applies custom width to column', () => {
      const columns: Column<TestData>[] = [
        {
          key: 'name',
          header: 'Name',
          accessor: (item) => item.name,
          width: '300px',
        },
      ];

      const { container } = render(
        <Table
          columns={columns}
          data={mockData}
          keyExtractor={(item) => item.id}
        />
      );
      const headerCell = container.querySelector('th') as HTMLElement;

      expect(headerCell.style.width).toBe('300px');
    });
  });

  describe('Sorting', () => {
    it('renders sort icons for sortable columns', () => {
      const { container } = render(
        <Table
          columns={mockColumns}
          data={mockData}
          keyExtractor={(item) => item.id}
        />
      );
      const sortIcons = container.querySelectorAll('svg');

      // Should have sort icons for sortable columns (Name and Status)
      expect(sortIcons.length).toBeGreaterThan(0);
    });

    it('does not render sort icon for non-sortable column', () => {
      const columns: Column<TestData>[] = [
        {
          key: 'name',
          header: 'Name',
          accessor: (item) => item.name,
          sortable: false,
        },
      ];

      const { container } = render(
        <Table
          columns={columns}
          data={[mockData[0]]}
          keyExtractor={(item) => item.id}
        />
      );
      const headerCell = container.querySelector('th');
      const sortIcon = headerCell?.querySelector('svg');

      expect(sortIcon).not.toBeInTheDocument();
    });

    it('shows ascending sort icon when sorted ascending', () => {
      render(
        <Table
          columns={mockColumns}
          data={mockData}
          keyExtractor={(item) => item.id}
          sortKey="name"
          sortDirection="asc"
        />
      );

      // ChevronUp icon should be present
      const { container } = render(
        <Table
          columns={mockColumns}
          data={mockData}
          keyExtractor={(item) => item.id}
          sortKey="name"
          sortDirection="asc"
        />
      );

      expect(container.querySelector('svg')).toBeInTheDocument();
    });

    it('shows descending sort icon when sorted descending', () => {
      const { container } = render(
        <Table
          columns={mockColumns}
          data={mockData}
          keyExtractor={(item) => item.id}
          sortKey="name"
          sortDirection="desc"
        />
      );

      expect(container.querySelector('svg')).toBeInTheDocument();
    });

    it('calls onSort when clicking sortable column header', async () => {
      const user = userEvent.setup();
      const onSort = vi.fn();

      render(
        <Table
          columns={mockColumns}
          data={mockData}
          keyExtractor={(item) => item.id}
          onSort={onSort}
        />
      );

      const nameHeader = screen.getByText('Name');
      await user.click(nameHeader);

      expect(onSort).toHaveBeenCalledWith('name');
    });

    it('does not call onSort when clicking non-sortable column', async () => {
      const user = userEvent.setup();
      const onSort = vi.fn();

      render(
        <Table
          columns={mockColumns}
          data={mockData}
          keyExtractor={(item) => item.id}
          onSort={onSort}
        />
      );

      const countHeader = screen.getByText('Count');
      await user.click(countHeader);

      expect(onSort).not.toHaveBeenCalled();
    });

    it('applies hover style to sortable headers', () => {
      const { container } = render(
        <Table
          columns={mockColumns}
          data={mockData}
          keyExtractor={(item) => item.id}
        />
      );
      const headers = container.querySelectorAll('th');
      const nameHeader = headers[0]; // 'Name' column (sortable)
      const countHeader = headers[2]; // 'Count' column (not sortable)

      expect(nameHeader).toHaveClass('cursor-pointer', 'hover:bg-gray-100');
      expect(countHeader).not.toHaveClass('cursor-pointer');
    });
  });

  describe('Row interactions', () => {
    it('calls onRowClick when row is clicked', async () => {
      const user = userEvent.setup();
      const onRowClick = vi.fn();

      render(
        <Table
          columns={mockColumns}
          data={mockData}
          keyExtractor={(item) => item.id}
          onRowClick={onRowClick}
        />
      );

      const firstRow = screen.getByText('Item 1').closest('tr')!;
      await user.click(firstRow);

      expect(onRowClick).toHaveBeenCalledWith(mockData[0]);
    });

    it('applies cursor pointer when onRowClick is provided', () => {
      const { container } = render(
        <Table
          columns={mockColumns}
          data={mockData}
          keyExtractor={(item) => item.id}
          onRowClick={vi.fn()}
        />
      );
      const row = container.querySelector('tbody tr');

      expect(row).toHaveClass('cursor-pointer');
    });

    it('does not apply cursor pointer when onRowClick is not provided', () => {
      const { container } = render(
        <Table
          columns={mockColumns}
          data={mockData}
          keyExtractor={(item) => item.id}
        />
      );
      const row = container.querySelector('tbody tr');

      expect(row).not.toHaveClass('cursor-pointer');
    });

    it('applies hover effect to rows', () => {
      const { container } = render(
        <Table
          columns={mockColumns}
          data={mockData}
          keyExtractor={(item) => item.id}
        />
      );
      const row = container.querySelector('tbody tr');

      expect(row).toHaveClass('hover:bg-primary-50/50');
    });
  });

  describe('Loading state', () => {
    it('shows loading spinner when loading', () => {
      render(
        <Table
          columns={mockColumns}
          data={mockData}
          keyExtractor={(item) => item.id}
          loading
        />
      );

      expect(screen.getByText('Loading data...')).toBeInTheDocument();
    });

    it('shows loading spinner with animation', () => {
      const { container } = render(
        <Table
          columns={mockColumns}
          data={mockData}
          keyExtractor={(item) => item.id}
          loading
        />
      );
      const spinner = container.querySelector('.animate-spin');

      expect(spinner).toBeInTheDocument();
    });

    it('does not show data when loading', () => {
      render(
        <Table
          columns={mockColumns}
          data={mockData}
          keyExtractor={(item) => item.id}
          loading
        />
      );

      expect(screen.queryByText('Item 1')).not.toBeInTheDocument();
      expect(screen.queryByText('Item 2')).not.toBeInTheDocument();
    });

    it('spans loading cell across all columns', () => {
      const { container } = render(
        <Table
          columns={mockColumns}
          data={mockData}
          keyExtractor={(item) => item.id}
          loading
        />
      );
      const loadingCell = container.querySelector('td');

      expect(loadingCell).toHaveAttribute('colSpan', '3');
    });
  });

  describe('Empty state', () => {
    it('shows default empty message when no data', () => {
      render(
        <Table
          columns={mockColumns}
          data={[]}
          keyExtractor={(item) => item.id}
        />
      );

      expect(screen.getByText('No data available')).toBeInTheDocument();
    });

    it('shows custom empty message', () => {
      render(
        <Table
          columns={mockColumns}
          data={[]}
          keyExtractor={(item) => item.id}
          emptyMessage="No orders found"
        />
      );

      expect(screen.getByText('No orders found')).toBeInTheDocument();
    });

    it('spans empty cell across all columns', () => {
      const { container } = render(
        <Table
          columns={mockColumns}
          data={[]}
          keyExtractor={(item) => item.id}
        />
      );
      const emptyCell = container.querySelector('td');

      expect(emptyCell).toHaveAttribute('colSpan', '3');
    });

    it('does not show empty message when data is present', () => {
      render(
        <Table
          columns={mockColumns}
          data={mockData}
          keyExtractor={(item) => item.id}
        />
      );

      expect(screen.queryByText('No data available')).not.toBeInTheDocument();
    });
  });

  describe('Styling', () => {
    it('has correct base table classes', () => {
      const { container } = render(
        <Table
          columns={mockColumns}
          data={mockData}
          keyExtractor={(item) => item.id}
        />
      );
      const wrapper = container.firstChild as HTMLElement;
      const table = container.querySelector('table');

      expect(wrapper).toHaveClass('overflow-x-auto', 'rounded-xl', 'border', 'border-gray-100');
      expect(table).toHaveClass('min-w-full');
    });

    it('has correct header styling', () => {
      const { container } = render(
        <Table
          columns={mockColumns}
          data={mockData}
          keyExtractor={(item) => item.id}
        />
      );
      const headerRow = container.querySelector('thead tr');

      expect(headerRow).toHaveClass('bg-gray-50/80', 'border-b', 'border-gray-100');
    });

    it('has correct header cell text styling', () => {
      const { container } = render(
        <Table
          columns={mockColumns}
          data={mockData}
          keyExtractor={(item) => item.id}
        />
      );
      const headerCell = container.querySelector('th');

      expect(headerCell).toHaveClass('text-xs', 'font-semibold', 'text-gray-600', 'uppercase', 'tracking-wider');
    });

    it('has transition classes on rows', () => {
      const { container } = render(
        <Table
          columns={mockColumns}
          data={mockData}
          keyExtractor={(item) => item.id}
        />
      );
      const row = container.querySelector('tbody tr');

      expect(row).toHaveClass('transition-colors', 'duration-150');
    });
  });

  describe('Complete examples', () => {
    it('renders complete table with sorting and row clicks', async () => {
      const user = userEvent.setup();
      const onSort = vi.fn();
      const onRowClick = vi.fn();

      render(
        <Table
          columns={mockColumns}
          data={mockData}
          keyExtractor={(item) => item.id}
          sortKey="name"
          sortDirection="asc"
          onSort={onSort}
          onRowClick={onRowClick}
        />
      );

      // Verify data rendering
      expect(screen.getByText('Item 1')).toBeInTheDocument();
      expect(screen.getByText('Active')).toBeInTheDocument();

      // Test sorting
      await user.click(screen.getByText('Status'));
      expect(onSort).toHaveBeenCalledWith('status');

      // Test row click
      const firstRow = screen.getByText('Item 1').closest('tr')!;
      await user.click(firstRow);
      expect(onRowClick).toHaveBeenCalledWith(mockData[0]);
    });
  });
});

describe('Pagination', () => {
  describe('Rendering', () => {
    it('renders pagination info text', () => {
      const { container } = render(
        <Pagination
          currentPage={1}
          totalPages={5}
          pageSize={10}
          totalItems={50}
          onPageChange={vi.fn()}
        />
      );

      expect(container.textContent).toContain('Showing');
      expect(container.textContent).toContain('results');
    });

    it('calculates correct start and end items', () => {
      const { container } = render(
        <Pagination
          currentPage={3}
          totalPages={5}
          pageSize={10}
          totalItems={50}
          onPageChange={vi.fn()}
        />
      );

      // Page 3: items 21-30 - check in container text
      expect(container.textContent).toContain('Showing');
      expect(container.textContent).toContain('21');
      expect(container.textContent).toContain('30');
    });

    it('shows correct end item on last page', () => {
      const { container } = render(
        <Pagination
          currentPage={3}
          totalPages={3}
          pageSize={20}
          totalItems={55}
          onPageChange={vi.fn()}
        />
      );

      // Page 3: items 41-55 (not 41-60)
      expect(container.textContent).toContain('Showing');
      expect(container.textContent).toContain('41');
      expect(container.textContent).toContain('55');
    });

    it('renders Previous button', () => {
      render(
        <Pagination
          currentPage={2}
          totalPages={5}
          pageSize={10}
          totalItems={50}
          onPageChange={vi.fn()}
        />
      );

      expect(screen.getByRole('button', { name: 'Previous' })).toBeInTheDocument();
    });

    it('renders Next button', () => {
      render(
        <Pagination
          currentPage={2}
          totalPages={5}
          pageSize={10}
          totalItems={50}
          onPageChange={vi.fn()}
        />
      );

      expect(screen.getByRole('button', { name: 'Next' })).toBeInTheDocument();
    });

    it('renders page number buttons', () => {
      render(
        <Pagination
          currentPage={1}
          totalPages={5}
          pageSize={10}
          totalItems={50}
          onPageChange={vi.fn()}
        />
      );

      expect(screen.getByRole('button', { name: '1' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: '2' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: '3' })).toBeInTheDocument();
    });
  });

  describe('Page size selector', () => {
    it('renders page size selector when onPageSizeChange provided', () => {
      render(
        <Pagination
          currentPage={1}
          totalPages={5}
          pageSize={10}
          totalItems={50}
          onPageChange={vi.fn()}
          onPageSizeChange={vi.fn()}
        />
      );

      expect(screen.getByRole('combobox')).toBeInTheDocument();
    });

    it('does not render page size selector when onPageSizeChange not provided', () => {
      render(
        <Pagination
          currentPage={1}
          totalPages={5}
          pageSize={10}
          totalItems={50}
          onPageChange={vi.fn()}
        />
      );

      expect(screen.queryByRole('combobox')).not.toBeInTheDocument();
    });

    it('renders default page size options', () => {
      render(
        <Pagination
          currentPage={1}
          totalPages={5}
          pageSize={10}
          totalItems={50}
          onPageChange={vi.fn()}
          onPageSizeChange={vi.fn()}
        />
      );

      expect(screen.getByText('10 per page')).toBeInTheDocument();
      expect(screen.getByText('20 per page')).toBeInTheDocument();
      expect(screen.getByText('50 per page')).toBeInTheDocument();
      expect(screen.getByText('100 per page')).toBeInTheDocument();
    });

    it('renders custom page size options', () => {
      render(
        <Pagination
          currentPage={1}
          totalPages={5}
          pageSize={25}
          totalItems={50}
          onPageChange={vi.fn()}
          onPageSizeChange={vi.fn()}
          pageSizeOptions={[25, 50, 75]}
        />
      );

      expect(screen.getByText('25 per page')).toBeInTheDocument();
      expect(screen.getByText('50 per page')).toBeInTheDocument();
      expect(screen.getByText('75 per page')).toBeInTheDocument();
    });

    it('calls onPageSizeChange when size is changed', async () => {
      const user = userEvent.setup();
      const onPageSizeChange = vi.fn();

      render(
        <Pagination
          currentPage={1}
          totalPages={5}
          pageSize={10}
          totalItems={50}
          onPageChange={vi.fn()}
          onPageSizeChange={onPageSizeChange}
        />
      );

      const select = screen.getByRole('combobox');
      await user.selectOptions(select, '20');

      expect(onPageSizeChange).toHaveBeenCalledWith(20);
    });
  });

  describe('Previous button', () => {
    it('is disabled on first page', () => {
      render(
        <Pagination
          currentPage={1}
          totalPages={5}
          pageSize={10}
          totalItems={50}
          onPageChange={vi.fn()}
        />
      );

      expect(screen.getByRole('button', { name: 'Previous' })).toBeDisabled();
    });

    it('is enabled on pages after first', () => {
      render(
        <Pagination
          currentPage={2}
          totalPages={5}
          pageSize={10}
          totalItems={50}
          onPageChange={vi.fn()}
        />
      );

      expect(screen.getByRole('button', { name: 'Previous' })).not.toBeDisabled();
    });

    it('calls onPageChange with previous page', async () => {
      const user = userEvent.setup();
      const onPageChange = vi.fn();

      render(
        <Pagination
          currentPage={3}
          totalPages={5}
          pageSize={10}
          totalItems={50}
          onPageChange={onPageChange}
        />
      );

      await user.click(screen.getByRole('button', { name: 'Previous' }));

      expect(onPageChange).toHaveBeenCalledWith(2);
    });
  });

  describe('Next button', () => {
    it('is disabled on last page', () => {
      render(
        <Pagination
          currentPage={5}
          totalPages={5}
          pageSize={10}
          totalItems={50}
          onPageChange={vi.fn()}
        />
      );

      expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled();
    });

    it('is enabled on pages before last', () => {
      render(
        <Pagination
          currentPage={4}
          totalPages={5}
          pageSize={10}
          totalItems={50}
          onPageChange={vi.fn()}
        />
      );

      expect(screen.getByRole('button', { name: 'Next' })).not.toBeDisabled();
    });

    it('calls onPageChange with next page', async () => {
      const user = userEvent.setup();
      const onPageChange = vi.fn();

      render(
        <Pagination
          currentPage={2}
          totalPages={5}
          pageSize={10}
          totalItems={50}
          onPageChange={onPageChange}
        />
      );

      await user.click(screen.getByRole('button', { name: 'Next' }));

      expect(onPageChange).toHaveBeenCalledWith(3);
    });
  });

  describe('Page number buttons', () => {
    it('shows max 5 page buttons', () => {
      render(
        <Pagination
          currentPage={1}
          totalPages={10}
          pageSize={10}
          totalItems={100}
          onPageChange={vi.fn()}
        />
      );

      const pageButtons = screen.getAllByRole('button').filter(
        button => !['Previous', 'Next'].includes(button.textContent || '')
      );

      expect(pageButtons.length).toBe(5);
    });

    it('shows all pages when total pages <= 5', () => {
      render(
        <Pagination
          currentPage={1}
          totalPages={3}
          pageSize={10}
          totalItems={30}
          onPageChange={vi.fn()}
        />
      );

      expect(screen.getByRole('button', { name: '1' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: '2' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: '3' })).toBeInTheDocument();
    });

    it('highlights current page', () => {
      render(
        <Pagination
          currentPage={3}
          totalPages={5}
          pageSize={10}
          totalItems={50}
          onPageChange={vi.fn()}
        />
      );

      const currentPageButton = screen.getByRole('button', { name: '3' });
      expect(currentPageButton).toHaveClass('bg-primary-600', 'text-white');
    });

    it('calls onPageChange when page number clicked', async () => {
      const user = userEvent.setup();
      const onPageChange = vi.fn();

      render(
        <Pagination
          currentPage={1}
          totalPages={5}
          pageSize={10}
          totalItems={50}
          onPageChange={onPageChange}
        />
      );

      await user.click(screen.getByRole('button', { name: '3' }));

      expect(onPageChange).toHaveBeenCalledWith(3);
    });

    it('shows pages 1-5 when on page 1', () => {
      render(
        <Pagination
          currentPage={1}
          totalPages={10}
          pageSize={10}
          totalItems={100}
          onPageChange={vi.fn()}
        />
      );

      expect(screen.getByRole('button', { name: '1' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: '5' })).toBeInTheDocument();
    });

    it('shows correct page range in middle', () => {
      render(
        <Pagination
          currentPage={5}
          totalPages={10}
          pageSize={10}
          totalItems={100}
          onPageChange={vi.fn()}
        />
      );

      // Should show pages 3, 4, 5, 6, 7 (currentPage - 2 to currentPage + 2)
      expect(screen.getByRole('button', { name: '3' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: '7' })).toBeInTheDocument();
    });

    it('shows last 5 pages when near end', () => {
      render(
        <Pagination
          currentPage={9}
          totalPages={10}
          pageSize={10}
          totalItems={100}
          onPageChange={vi.fn()}
        />
      );

      // Should show pages 6, 7, 8, 9, 10
      expect(screen.getByRole('button', { name: '6' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: '10' })).toBeInTheDocument();
    });
  });

  describe('Complete examples', () => {
    it('renders complete pagination with all features', async () => {
      const user = userEvent.setup();
      const onPageChange = vi.fn();
      const onPageSizeChange = vi.fn();

      const { container } = render(
        <Pagination
          currentPage={2}
          totalPages={10}
          pageSize={20}
          totalItems={200}
          onPageChange={onPageChange}
          onPageSizeChange={onPageSizeChange}
          pageSizeOptions={[20, 50, 100]}
        />
      );

      // Verify info text
      expect(container.textContent).toContain('Showing');
      expect(container.textContent).toContain('21');
      expect(container.textContent).toContain('40');
      expect(container.textContent).toContain('200');

      // Test page size change
      await user.selectOptions(screen.getByRole('combobox'), '50');
      expect(onPageSizeChange).toHaveBeenCalledWith(50);

      // Test navigation
      await user.click(screen.getByRole('button', { name: 'Next' }));
      expect(onPageChange).toHaveBeenCalledWith(3);

      await user.click(screen.getByRole('button', { name: 'Previous' }));
      expect(onPageChange).toHaveBeenCalledWith(1);
    });

    it('renders single page correctly', () => {
      const { container } = render(
        <Pagination
          currentPage={1}
          totalPages={1}
          pageSize={10}
          totalItems={5}
          onPageChange={vi.fn()}
        />
      );

      expect(container.textContent).toContain('Showing');
      expect(container.textContent).toContain('5');
      expect(screen.getByRole('button', { name: 'Previous' })).toBeDisabled();
      expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled();
    });
  });
});
