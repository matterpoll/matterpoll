import React from 'react';
import {fireEvent, render, screen} from '@testing-library/react';

import DefaultSettings from '@/components/admin_settings/default_settings';

import type {DefaultSettingsValue} from '@/types/poll';

describe('components/admin_settings/DefaultSettings', () => {
    const baseProps = {
        id: 'test id',
        value: {
            anonymous: false,
            anonymousCreator: true,
            progress: false,
            publicAddOption: true,
        },
        label: 'test label',
        disabled: false,
        setByEnv: false,
        config: {},
        license: {},
        helpText: null,
        onChange: jest.fn(),
        registerSaveAction: jest.fn(),
        setSaveNeeded: jest.fn(),
        unRegisterSaveAction: jest.fn(),
    };

    const checkboxFor = (title: string) => {
        // Each setting renders its title next to its own checkbox.
        const row = screen.getByText(title).closest('.row') as HTMLElement;
        return row.querySelector('input[type="checkbox"]') as HTMLInputElement;
    };

    test('should render one setting per poll default', () => {
        render(<DefaultSettings {...baseProps}/>);

        expect(screen.getByText('Anonymous')).toBeInTheDocument();
        expect(screen.getByText('Anonymous Creator')).toBeInTheDocument();
        expect(screen.getByText('Progress')).toBeInTheDocument();
        expect(screen.getByText('Public Add Option')).toBeInTheDocument();
        expect(screen.getAllByRole('checkbox')).toHaveLength(4);
    });

    test('should seed each checkbox from the matching key of the value prop', () => {
        render(<DefaultSettings {...baseProps}/>);

        expect(checkboxFor('Anonymous')).not.toBeChecked();
        expect(checkboxFor('Anonymous Creator')).toBeChecked();
        expect(checkboxFor('Progress')).not.toBeChecked();
        expect(checkboxFor('Public Add Option')).toBeChecked();
    });

    test('should render every checkbox checked when all options are true', () => {
        const value: DefaultSettingsValue = {
            anonymous: true,
            anonymousCreator: true,
            progress: true,
            publicAddOption: true,
        };
        render(
            <DefaultSettings
                {...baseProps}
                value={value}
            />,
        );

        screen.getAllByRole('checkbox').forEach((checkbox) => expect(checkbox).toBeChecked());
    });

    test('should render every checkbox unchecked when all options are false', () => {
        const value: DefaultSettingsValue = {
            anonymous: false,
            anonymousCreator: false,
            progress: false,
            publicAddOption: false,
        };
        render(
            <DefaultSettings
                {...baseProps}
                value={value}
            />,
        );

        screen.getAllByRole('checkbox').forEach((checkbox) => expect(checkbox).not.toBeChecked());
    });

    test('should report the whole settings object, not just the changed key', () => {
        const onChange = jest.fn();
        const setSaveNeeded = jest.fn();
        render(
            <DefaultSettings
                {...baseProps}
                onChange={onChange}
                setSaveNeeded={setSaveNeeded}
            />,
        );

        fireEvent.click(checkboxFor('Progress'));

        expect(onChange).toHaveBeenCalledWith('test id', {
            anonymous: false,
            anonymousCreator: true,
            progress: true,
            publicAddOption: true,
        });
        expect(setSaveNeeded).toHaveBeenCalled();
    });

    test('should accumulate changes across several settings', () => {
        const onChange = jest.fn();
        render(
            <DefaultSettings
                {...baseProps}
                onChange={onChange}
            />,
        );

        fireEvent.click(checkboxFor('Progress'));
        fireEvent.click(checkboxFor('Public Add Option'));

        expect(onChange).toHaveBeenLastCalledWith('test id', {
            anonymous: false,
            anonymousCreator: true,
            progress: true,
            publicAddOption: false,
        });
    });
});
