import React from 'react';

import type {AdminConfig, ClientLicense} from '@mattermost/types/config';

import type {DefaultSettingsValue} from '@/types/poll';

import DefaultSetting from './default_setting';

type Props = {
    id: string;
    label: string;
    helpText?: React.ReactNode;
    value?: DefaultSettingsValue;
    disabled: boolean;
    config: Partial<AdminConfig>;
    license: ClientLicense;
    setByEnv: boolean;
    onChange: (id: string, value: DefaultSettingsValue) => void;
    registerSaveAction: (action: () => Promise<unknown>) => void;
    setSaveNeeded: () => void;
    unRegisterSaveAction: (action: () => Promise<unknown>) => void;
};

export default class DefaultSettings extends React.Component<Props> {
    private settings: DefaultSettingsValue;

    constructor(props: Props) {
        super(props);

        this.settings = {
            ...props.value,
        };
    }

    handleChange = (name: string, value: boolean) => {
        this.settings = {...this.settings, [name]: value};
        this.props.onChange(this.props.id, this.settings);
        this.props.setSaveNeeded();
    };

    render() {
        return (
            <div>
                <DefaultSetting
                    name={'anonymous'}
                    title={'Anonymous'}
                    label={'Don\'t show who voted for what when the poll ends'}
                    value={this.settings.anonymous}
                    onChange={this.handleChange}
                />
                <DefaultSetting
                    name={'anonymousCreator'}
                    title={'Anonymous Creator'}
                    label={'Don\'t show author of the poll'}
                    value={this.settings.anonymousCreator}
                    onChange={this.handleChange}
                />
                <DefaultSetting
                    name={'progress'}
                    title={'Progress'}
                    label={'During the poll, show how many votes each answer option got'}
                    value={this.settings.progress}
                    onChange={this.handleChange}
                />
                <DefaultSetting
                    name={'publicAddOption'}
                    title={'Public Add Option'}
                    label={'Allow all users to add additional options'}
                    value={this.settings.publicAddOption}
                    onChange={this.handleChange}
                />
            </div>
        );
    }
}
