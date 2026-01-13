import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Input, SearchInput, Select } from './Input';
import { Mail, User } from 'lucide-react';

describe('Input', () => {
  describe('Rendering', () => {
    it('renders with default props', () => {
      render(<Input />);
      const input = screen.getByRole('textbox');

      expect(input).toBeInTheDocument();
    });

    it('renders with placeholder', () => {
      render(<Input placeholder="Enter text" />);
      const input = screen.getByPlaceholderText('Enter text');

      expect(input).toBeInTheDocument();
    });

    it('renders with value', () => {
      render(<Input value="Test value" onChange={vi.fn()} />);
      const input = screen.getByDisplayValue('Test value');

      expect(input).toBeInTheDocument();
    });

    it('forwards ref to input element', () => {
      const ref = vi.fn();
      render(<Input ref={ref} />);

      expect(ref).toHaveBeenCalledWith(expect.any(HTMLInputElement));
    });
  });

  describe('Label', () => {
    it('renders label when provided', () => {
      render(<Input label="Email Address" />);

      expect(screen.getByText('Email Address')).toBeInTheDocument();
    });

    it('does not render label when not provided', () => {
      const { container } = render(<Input />);
      const label = container.querySelector('label');

      expect(label).not.toBeInTheDocument();
    });

    it('renders required indicator when required', () => {
      render(<Input label="Email" required />);

      expect(screen.getByText('*')).toBeInTheDocument();
      expect(screen.getByText('*')).toHaveClass('text-error-500');
    });

    it('does not render required indicator when not required', () => {
      render(<Input label="Email" />);
      const asterisks = Array.from(document.querySelectorAll('.text-error-500'));

      expect(asterisks.length).toBe(0);
    });

    it('associates label with input', () => {
      render(<Input label="Username" id="username-input" />);
      const input = screen.getByRole('textbox');
      const label = screen.getByText('Username');

      expect(label.tagName).toBe('LABEL');
      expect(input).toBeInTheDocument();
    });
  });

  describe('Size variants', () => {
    it('renders small size', () => {
      render(<Input size="sm" />);
      const input = screen.getByRole('textbox');

      expect(input).toHaveClass('px-3', 'py-1.5', 'text-sm');
    });

    it('renders medium size by default', () => {
      render(<Input />);
      const input = screen.getByRole('textbox');

      expect(input).toHaveClass('px-3', 'py-2', 'text-sm');
    });

    it('renders large size', () => {
      render(<Input size="lg" />);
      const input = screen.getByRole('textbox');

      expect(input).toHaveClass('px-4', 'py-3', 'text-base');
    });
  });

  describe('Error state', () => {
    it('renders error message', () => {
      render(<Input error="This field is required" />);

      expect(screen.getByText('This field is required')).toBeInTheDocument();
    });

    it('applies error border color', () => {
      render(<Input error="Error message" />);
      const input = screen.getByRole('textbox');

      expect(input).toHaveClass('border-error-500');
    });

    it('shows error icon when error is present', () => {
      const { container } = render(<Input error="Error message" />);
      const errorIcon = container.querySelector('.text-error-500');

      expect(errorIcon).toBeInTheDocument();
    });

    it('does not show error icon when no error', () => {
      render(<Input />);
      const input = screen.getByRole('textbox');

      expect(input).not.toHaveClass('border-error-500');
    });

    it('applies error text color to error message', () => {
      render(<Input error="Error message" />);
      const errorText = screen.getByText('Error message');

      expect(errorText).toHaveClass('text-error-600');
    });
  });

  describe('Hint text', () => {
    it('renders hint when provided', () => {
      render(<Input hint="Enter your email address" />);

      expect(screen.getByText('Enter your email address')).toBeInTheDocument();
    });

    it('applies gray color to hint text', () => {
      render(<Input hint="Helpful hint" />);
      const hintText = screen.getByText('Helpful hint');

      expect(hintText).toHaveClass('text-gray-500');
    });

    it('shows error instead of hint when both provided', () => {
      render(<Input error="Error message" hint="Hint text" />);

      expect(screen.getByText('Error message')).toBeInTheDocument();
      expect(screen.queryByText('Hint text')).not.toBeInTheDocument();
    });

    it('does not render hint or error text when neither provided', () => {
      const { container } = render(<Input />);
      const helpText = container.querySelector('.mt-1.text-sm');

      expect(helpText).not.toBeInTheDocument();
    });
  });

  describe('Icons', () => {
    it('renders left icon', () => {
      render(<Input leftIcon={<Mail data-testid="left-icon" />} />);

      expect(screen.getByTestId('left-icon')).toBeInTheDocument();
    });

    it('renders right icon', () => {
      render(<Input rightIcon={<User data-testid="right-icon" />} />);

      expect(screen.getByTestId('right-icon')).toBeInTheDocument();
    });

    it('renders both left and right icons', () => {
      render(
        <Input
          leftIcon={<Mail data-testid="left-icon" />}
          rightIcon={<User data-testid="right-icon" />}
        />
      );

      expect(screen.getByTestId('left-icon')).toBeInTheDocument();
      expect(screen.getByTestId('right-icon')).toBeInTheDocument();
    });

    it('adjusts padding when left icon present', () => {
      render(<Input leftIcon={<Mail />} />);
      const input = screen.getByRole('textbox');

      expect(input).toHaveClass('pl-10');
    });

    it('adjusts padding when right icon present', () => {
      render(<Input rightIcon={<User />} />);
      const input = screen.getByRole('textbox');

      expect(input).toHaveClass('pr-10');
    });

    it('shows error icon instead of right icon when error present', () => {
      render(
        <Input
          error="Error"
          rightIcon={<User data-testid="right-icon" />}
        />
      );

      expect(screen.queryByTestId('right-icon')).not.toBeInTheDocument();
    });
  });

  describe('Disabled state', () => {
    it('disables input when disabled prop is true', () => {
      render(<Input disabled />);
      const input = screen.getByRole('textbox');

      expect(input).toBeDisabled();
    });

    it('applies disabled styles', () => {
      render(<Input disabled />);
      const input = screen.getByRole('textbox');

      expect(input).toHaveClass('disabled:bg-gray-50', 'disabled:text-gray-500', 'disabled:cursor-not-allowed');
    });

    it('is not disabled by default', () => {
      render(<Input />);
      const input = screen.getByRole('textbox');

      expect(input).not.toBeDisabled();
    });
  });

  describe('Input types', () => {
    it('renders text input by default', () => {
      render(<Input />);
      const input = screen.getByRole('textbox');

      // Text inputs have an implicit type, checking role is sufficient
      expect(input).toBeInTheDocument();
    });

    it('renders email input', () => {
      render(<Input type="email" />);
      const input = document.querySelector('input[type="email"]');

      expect(input).toBeInTheDocument();
    });

    it('renders password input', () => {
      render(<Input type="password" />);
      const input = document.querySelector('input[type="password"]');

      expect(input).toBeInTheDocument();
    });

    it('renders number input', () => {
      render(<Input type="number" />);
      const input = screen.getByRole('spinbutton');

      expect(input).toBeInTheDocument();
    });
  });

  describe('User interactions', () => {
    it('calls onChange when input value changes', async () => {
      const user = userEvent.setup();
      const onChange = vi.fn();
      render(<Input onChange={onChange} />);

      const input = screen.getByRole('textbox');
      await user.type(input, 'Hello');

      expect(onChange).toHaveBeenCalled();
    });

    it('updates value on user input', async () => {
      const user = userEvent.setup();
      render(<Input />);

      const input = screen.getByRole('textbox') as HTMLInputElement;
      await user.type(input, 'Test input');

      expect(input.value).toBe('Test input');
    });

    it('does not call onChange when disabled', async () => {
      const user = userEvent.setup();
      const onChange = vi.fn();
      render(<Input disabled onChange={onChange} />);

      const input = screen.getByRole('textbox');
      await user.type(input, 'Hello');

      expect(onChange).not.toHaveBeenCalled();
    });
  });

  describe('Styling', () => {
    it('has correct base classes', () => {
      render(<Input />);
      const input = screen.getByRole('textbox');

      expect(input).toHaveClass('block', 'w-full', 'rounded-lg', 'border', 'bg-white');
    });

    it('has transition classes', () => {
      render(<Input />);
      const input = screen.getByRole('textbox');

      expect(input).toHaveClass('transition-colors', 'duration-200');
    });

    it('has focus ring classes', () => {
      render(<Input />);
      const input = screen.getByRole('textbox');

      expect(input).toHaveClass('focus:outline-none', 'focus:ring-2', 'focus:ring-offset-0');
    });

    it('has default border color when no error', () => {
      render(<Input />);
      const input = screen.getByRole('textbox');

      expect(input).toHaveClass('border-gray-300');
    });

    it('applies custom className', () => {
      render(<Input className="custom-input" />);
      const input = screen.getByRole('textbox');

      expect(input).toHaveClass('custom-input');
    });
  });

  describe('Complete examples', () => {
    it('renders complete form input with all props', () => {
      render(
        <Input
          label="Email Address"
          type="email"
          placeholder="you@example.com"
          hint="We'll never share your email"
          required
          leftIcon={<Mail data-testid="icon" />}
          size="lg"
        />
      );

      expect(screen.getByText('Email Address')).toBeInTheDocument();
      expect(screen.getByPlaceholderText('you@example.com')).toBeInTheDocument();
      expect(screen.getByText("We'll never share your email")).toBeInTheDocument();
      expect(screen.getByText('*')).toBeInTheDocument();
      expect(screen.getByTestId('icon')).toBeInTheDocument();
    });

    it('renders input with error state', () => {
      render(
        <Input
          label="Username"
          error="Username is already taken"
          value="testuser"
          onChange={vi.fn()}
        />
      );

      expect(screen.getByText('Username')).toBeInTheDocument();
      expect(screen.getByText('Username is already taken')).toBeInTheDocument();
      expect(screen.getByDisplayValue('testuser')).toHaveClass('border-error-500');
    });
  });
});

describe('SearchInput', () => {
  describe('Rendering', () => {
    it('renders with search icon', () => {
      const { container } = render(<SearchInput />);
      const searchIcon = container.querySelector('svg');

      expect(searchIcon).toBeInTheDocument();
    });

    it('renders with search type', () => {
      render(<SearchInput />);
      const input = document.querySelector('input[type="search"]');

      expect(input).toBeInTheDocument();
    });

    it('renders with default placeholder', () => {
      render(<SearchInput />);

      expect(screen.getByPlaceholderText('Search...')).toBeInTheDocument();
    });

    it('renders with custom placeholder', () => {
      render(<SearchInput placeholder="Search orders..." />);

      expect(screen.getByPlaceholderText('Search orders...')).toBeInTheDocument();
    });

    it('forwards ref to input element', () => {
      const ref = vi.fn();
      render(<SearchInput ref={ref} />);

      expect(ref).toHaveBeenCalledWith(expect.any(HTMLInputElement));
    });
  });

  describe('Search functionality', () => {
    it('calls onSearch when Enter key is pressed', async () => {
      const user = userEvent.setup();
      const onSearch = vi.fn();
      render(<SearchInput onSearch={onSearch} />);

      const input = screen.getByPlaceholderText('Search...');
      await user.type(input, 'test query');
      await user.keyboard('{Enter}');

      expect(onSearch).toHaveBeenCalledWith('test query');
    });

    it('does not call onSearch for other keys', async () => {
      const user = userEvent.setup();
      const onSearch = vi.fn();
      render(<SearchInput onSearch={onSearch} />);

      const input = screen.getByPlaceholderText('Search...');
      await user.type(input, 'test');
      await user.keyboard('{Escape}');
      await user.keyboard('{Tab}');

      expect(onSearch).not.toHaveBeenCalled();
    });

    it('calls onSearch with current input value', async () => {
      const user = userEvent.setup();
      const onSearch = vi.fn();
      render(<SearchInput onSearch={onSearch} />);

      const input = screen.getByPlaceholderText('Search...');
      await user.type(input, 'search term{Enter}');

      expect(onSearch).toHaveBeenCalledWith('search term');
    });

    it('does not error when onSearch not provided', async () => {
      const user = userEvent.setup();
      render(<SearchInput />);

      const input = screen.getByPlaceholderText('Search...');
      await user.type(input, 'test{Enter}');

      // Should not throw error
      expect(input).toBeInTheDocument();
    });
  });

  describe('Input props passthrough', () => {
    it('passes through label prop', () => {
      render(<SearchInput label="Search Orders" />);

      expect(screen.getByText('Search Orders')).toBeInTheDocument();
    });

    it('passes through error prop', () => {
      render(<SearchInput error="Search failed" />);

      expect(screen.getByText('Search failed')).toBeInTheDocument();
    });

    it('passes through hint prop', () => {
      render(<SearchInput hint="Enter at least 3 characters" />);

      expect(screen.getByText('Enter at least 3 characters')).toBeInTheDocument();
    });

    it('passes through size prop', () => {
      render(<SearchInput size="lg" />);
      const input = screen.getByPlaceholderText('Search...');

      expect(input).toHaveClass('px-4', 'py-3', 'text-base');
    });

    it('passes through disabled prop', () => {
      render(<SearchInput disabled />);
      const input = screen.getByPlaceholderText('Search...');

      expect(input).toBeDisabled();
    });
  });

  describe('Complete examples', () => {
    it('renders complete search input', async () => {
      const user = userEvent.setup();
      const onSearch = vi.fn();
      render(
        <SearchInput
          label="Search Products"
          hint="Press Enter to search"
          onSearch={onSearch}
          size="lg"
        />
      );

      expect(screen.getByText('Search Products')).toBeInTheDocument();
      expect(screen.getByText('Press Enter to search')).toBeInTheDocument();

      const input = screen.getByPlaceholderText('Search...');
      await user.type(input, 'laptop{Enter}');

      expect(onSearch).toHaveBeenCalledWith('laptop');
    });
  });
});

describe('Select', () => {
  const mockOptions = [
    { value: 'option1', label: 'Option 1' },
    { value: 'option2', label: 'Option 2' },
    { value: 'option3', label: 'Option 3' },
  ];

  describe('Rendering', () => {
    it('renders select with options', () => {
      render(<Select options={mockOptions} />);
      const select = screen.getByRole('combobox');

      expect(select).toBeInTheDocument();
      expect(screen.getByText('Option 1')).toBeInTheDocument();
      expect(screen.getByText('Option 2')).toBeInTheDocument();
      expect(screen.getByText('Option 3')).toBeInTheDocument();
    });

    it('renders all option values correctly', () => {
      render(<Select options={mockOptions} />);
      const options = screen.getAllByRole('option');

      expect(options[0]).toHaveValue('option1');
      expect(options[1]).toHaveValue('option2');
      expect(options[2]).toHaveValue('option3');
    });

    it('forwards ref to select element', () => {
      const ref = vi.fn();
      render(<Select ref={ref} options={mockOptions} />);

      expect(ref).toHaveBeenCalledWith(expect.any(HTMLSelectElement));
    });
  });

  describe('Label', () => {
    it('renders label when provided', () => {
      render(<Select label="Choose an option" options={mockOptions} />);

      expect(screen.getByText('Choose an option')).toBeInTheDocument();
    });

    it('does not render label when not provided', () => {
      const { container } = render(<Select options={mockOptions} />);
      const label = container.querySelector('label');

      expect(label).not.toBeInTheDocument();
    });

    it('renders required indicator when required', () => {
      render(<Select label="Category" options={mockOptions} required />);

      expect(screen.getByText('*')).toBeInTheDocument();
      expect(screen.getByText('*')).toHaveClass('text-error-500');
    });

    it('does not render required indicator when not required', () => {
      render(<Select label="Category" options={mockOptions} />);
      const asterisks = Array.from(document.querySelectorAll('.text-error-500'));

      expect(asterisks.length).toBe(0);
    });
  });

  describe('Error state', () => {
    it('renders error message', () => {
      render(<Select options={mockOptions} error="Please select an option" />);

      expect(screen.getByText('Please select an option')).toBeInTheDocument();
    });

    it('applies error border color', () => {
      render(<Select options={mockOptions} error="Error message" />);
      const select = screen.getByRole('combobox');

      expect(select).toHaveClass('border-error-500');
    });

    it('applies error text color to error message', () => {
      render(<Select options={mockOptions} error="Error message" />);
      const errorText = screen.getByText('Error message');

      expect(errorText).toHaveClass('text-error-600');
    });
  });

  describe('Hint text', () => {
    it('renders hint when provided', () => {
      render(<Select options={mockOptions} hint="Choose your preferred option" />);

      expect(screen.getByText('Choose your preferred option')).toBeInTheDocument();
    });

    it('applies gray color to hint text', () => {
      render(<Select options={mockOptions} hint="Helpful hint" />);
      const hintText = screen.getByText('Helpful hint');

      expect(hintText).toHaveClass('text-gray-500');
    });

    it('shows error instead of hint when both provided', () => {
      render(
        <Select
          options={mockOptions}
          error="Error message"
          hint="Hint text"
        />
      );

      expect(screen.getByText('Error message')).toBeInTheDocument();
      expect(screen.queryByText('Hint text')).not.toBeInTheDocument();
    });
  });

  describe('User interactions', () => {
    it('calls onChange when option is selected', async () => {
      const user = userEvent.setup();
      const onChange = vi.fn();
      render(<Select options={mockOptions} onChange={onChange} />);

      const select = screen.getByRole('combobox');
      await user.selectOptions(select, 'option2');

      expect(onChange).toHaveBeenCalled();
    });

    it('updates selected value', async () => {
      const user = userEvent.setup();
      render(<Select options={mockOptions} />);

      const select = screen.getByRole('combobox') as HTMLSelectElement;
      await user.selectOptions(select, 'option3');

      expect(select.value).toBe('option3');
    });

    it('does not call onChange when disabled', async () => {
      const user = userEvent.setup();
      const onChange = vi.fn();
      render(<Select options={mockOptions} disabled onChange={onChange} />);

      const select = screen.getByRole('combobox');
      await user.selectOptions(select, 'option2');

      expect(onChange).not.toHaveBeenCalled();
    });
  });

  describe('Disabled state', () => {
    it('disables select when disabled prop is true', () => {
      render(<Select options={mockOptions} disabled />);
      const select = screen.getByRole('combobox');

      expect(select).toBeDisabled();
    });

    it('applies disabled styles', () => {
      render(<Select options={mockOptions} disabled />);
      const select = screen.getByRole('combobox');

      expect(select).toHaveClass('disabled:bg-gray-50', 'disabled:text-gray-500', 'disabled:cursor-not-allowed');
    });
  });

  describe('Styling', () => {
    it('has correct base classes', () => {
      render(<Select options={mockOptions} />);
      const select = screen.getByRole('combobox');

      expect(select).toHaveClass('block', 'w-full', 'rounded-lg', 'border', 'bg-white');
    });

    it('has transition classes', () => {
      render(<Select options={mockOptions} />);
      const select = screen.getByRole('combobox');

      expect(select).toHaveClass('transition-colors', 'duration-200');
    });

    it('has focus ring classes', () => {
      render(<Select options={mockOptions} />);
      const select = screen.getByRole('combobox');

      expect(select).toHaveClass('focus:outline-none', 'focus:ring-2', 'focus:ring-offset-0');
    });

    it('applies custom className', () => {
      render(<Select options={mockOptions} className="custom-select" />);
      const select = screen.getByRole('combobox');

      expect(select).toHaveClass('custom-select');
    });
  });

  describe('Complete examples', () => {
    it('renders complete select with all props', () => {
      const options = [
        { value: 'pending', label: 'Pending' },
        { value: 'completed', label: 'Completed' },
        { value: 'cancelled', label: 'Cancelled' },
      ];

      render(
        <Select
          label="Order Status"
          options={options}
          hint="Filter orders by status"
          required
        />
      );

      expect(screen.getByText('Order Status')).toBeInTheDocument();
      expect(screen.getByText('Filter orders by status')).toBeInTheDocument();
      expect(screen.getByText('*')).toBeInTheDocument();
      expect(screen.getByText('Pending')).toBeInTheDocument();
      expect(screen.getByText('Completed')).toBeInTheDocument();
      expect(screen.getByText('Cancelled')).toBeInTheDocument();
    });

    it('renders select with error state', () => {
      render(
        <Select
          label="Category"
          options={mockOptions}
          error="Category is required"
        />
      );

      expect(screen.getByText('Category')).toBeInTheDocument();
      expect(screen.getByText('Category is required')).toBeInTheDocument();
      expect(screen.getByRole('combobox')).toHaveClass('border-error-500');
    });
  });
});
