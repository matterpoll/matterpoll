import React from 'react';

type Props = {
    name: string;
    title: string;
    label: string;
    value?: boolean;
    onChange: (name: string, value: boolean) => void;
};

export default class DefaultSetting extends React.Component<Props> {
    handleChange = (e: React.MouseEvent<HTMLInputElement>) => {
        this.props.onChange(this.props.name, e.currentTarget.checked);
    };

    render() {
        return (
            <div
                className='row'
                style={styles.row}
            >
                <div
                    className='col-xs-12 col-sm-4'
                    style={styles.label}
                >
                    <strong>{this.props.title}</strong>
                </div>
                <div className='col-xs-12 col-sm-8'>
                    <div className='checkbox'>
                        <label>
                            <input
                                type='checkbox'
                                defaultChecked={this.props.value}
                                onClick={this.handleChange}
                            />
                            <span>{this.props.label}</span>
                        </label>
                    </div>
                </div>
            </div>
        );
    }
}

const styles: Record<string, React.CSSProperties> = {
    label: {
        marginTop: '6px',
    },
};
