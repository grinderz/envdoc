package logging

type Config struct {
	// Level of the logging.
	Level string `env:"LEVEL" envDefault:"info"`
	// Format of the logging.
	Format string `env:"FORMAT" envDefault:"json"`
}
