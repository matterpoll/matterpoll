import React from 'react';
import {fireEvent, render, screen} from '@testing-library/react';

import DefaultSetting from '@/components/admin_settings/default_setting';

describe('components/admin_settings/DefaultSetting', () => {
    const baseProps = {
        name: 'test name',
        title: 'test title',
        label: 'test label',
        value: false,
        onChange: jest.fn(),
    };

    test('should render the title and label', () => {
        render(<DefaultSetting {...baseProps}/>);

        expect(screen.getByText('test title')).toBeInTheDocument();
        expect(screen.getByText('test label')).toBeInTheDocument();
    });

    test('should keep the grid class names the admin console styles settings by', () => {
        const {container} = render(<DefaultSetting {...baseProps}/>);

        expect(container.querySelector('.row > .col-xs-12.col-sm-4')).toHaveTextContent('test title');
        expect(container.querySelector('.row > .col-xs-12.col-sm-8 .checkbox')).toBeInTheDocument();
    });

    test('should render an unchecked checkbox for a false value', () => {
        render(<DefaultSetting {...baseProps}/>);

        expect(screen.getByRole('checkbox')).not.toBeChecked();
    });

    test('should render a checked checkbox for a true value', () => {
        render(
            <DefaultSetting
                {...baseProps}
                value={true}
            />,
        );

        expect(screen.getByRole('checkbox')).toBeChecked();
    });

    test('should report the new value under its own name when toggled on', () => {
        const onChange = jest.fn();
        render(
            <DefaultSetting
                {...baseProps}
                onChange={onChange}
            />,
        );

        fireEvent.click(screen.getByRole('checkbox'));

        expect(onChange).toHaveBeenCalledWith('test name', true);
    });

    test('should report the new value under its own name when toggled off', () => {
        const onChange = jest.fn();
        render(
            <DefaultSetting
                {...baseProps}
                value={true}
                onChange={onChange}
            />,
        );

        fireEvent.click(screen.getByRole('checkbox'));

        expect(onChange).toHaveBeenCalledWith('test name', false);
    });
});
