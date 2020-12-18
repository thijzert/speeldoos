/* eslint-disable no-undef */
/*
 *   gulp compile --production   >> productie versie. (Geen inline JS/CSS sourcemaps)
 *   gulp compile --develop
 */

let sassPaths = [ ".", "..", "node_modules/sass-mq" ];

let svgConfig = {
	"svg": {
		"xmlDeclaration": false,
		"doctypeDeclaration": false
	},
	"mode": {
		"symbol": true,
		"inline": true
	}
};

let gulp            = require("gulp");
let sourcemaps      = require("gulp-sourcemaps");
let minify          = require('gulp-minify');
let sass            = require("gulp-sass");
let postcss         = require("gulp-postcss");
let autoprefixer    = require("autoprefixer");
let postCssMqPacker = require("css-mqpacker");
let postCssReporter = require("postcss-reporter");
let postCssPxtorem  = require('postcss-pxtorem');
let postCssSvg      = require("postcss-inline-svg");
let webpackStream   = require('webpack-stream');
let webpack         = require('webpack');
let gulpif          = require('gulp-if');
let argv            = require('yargs').argv;
let named           = require('vinyl-named')
let svgSprite       = require('gulp-svg-sprite');
let babel           = require('gulp-babel');
let changed         = require('gulp-changed');
let csso            = require('gulp-csso');
let cached          = require('gulp-cached');
let dependents      = require('gulp-dependents');


// Compile to ES5 (Ancient browsers)
const compileToES5 = function()
{
	let jsEntrypoints = [
		"pkg/web/assets/src/js/speeldoos.js",
	];

	let webpackConfig = {
		mode: "production",
		output: {
			filename: "[name].js",
			publicPath: "/ancient-js/",
		},
		module: {
			rules: [
				{
					test: /\.js$/,
					exclude: /node_modules/,
					use: {
						loader: "babel-loader",
						options: {
							sourceMap: "inline",
							presets: [
								[
									"@babel/preset-env",
									{
										debug: false,
										targets: {
											browsers: [
												"defaults",
												"ie >=11"
											]
										},
										useBuiltIns: "usage",
										modules: false,
										corejs: 3
									}
								]
							]
						}
					}
				}
			]
		},
	};

	if ( !argv.production )
	{
		webpackConfig.mode = "development";
	}

	return gulp.src( jsEntrypoints ).
		pipe(gulpif(!argv.production, sourcemaps.init())).
		pipe(named()).
		pipe(webpackStream( webpackConfig, webpack )).
		pipe(gulpif(!argv.production, sourcemaps.write())).
		pipe(gulp.dest("pkg/web/assets/dist/ancient-js/"));
};

// Compile to ES2015+ (Modules)
const compileToES2015 = function()
{
	const localBabel = babel({
		"presets": [
			[
				"@babel/preset-env",
				{
					"targets": {
						"browsers": [ ">2%" ]
					},
					"useBuiltIns": false,
					"modules": false
				}
			]
		],
		"sourceMap" : "inline"
	});


	return gulp.src([ "pkg/web/assets/src/js/*.js" ]).
		pipe(changed("pkg/web/assets/dist/js/", { hasChanged: changed.compareContents })).
		pipe(gulpif(!argv.production, sourcemaps.init())).
		pipe(localBabel).
		pipe(gulpif(!argv.production, minify({
			noSource: true,
			mangle: false,
			ext: ".js",
		}))).
		pipe(gulpif(!argv.production, sourcemaps.write("."))).
		pipe(gulp.dest("pkg/web/assets/dist/js/"));
};


const compileBothScripts = gulp.series( compileToES2015, compileToES5 );

const watchScript = function()
{
	gulp.watch( [ "pkg/web/assets/src/js/*.js" ], compileBothScripts );
};

// Compile CSS
function compileStyle()
{
	const localPostCSS = postcss([
		postCssSvg({ paths: ["pkg/web/assets/src/svg"] }),
		autoprefixer({ grid: true }),
		postCssMqPacker(),
		postCssPxtorem({
			mediaQuery: false,
			propList: [ 'font', 'font-size', 'line-height', 'letter-spacing', 'min-width', 'max-width', 'width', 'height', 'padding*', 'margin*', 'padding', 'margin' ],
			minPixelValue: 16
		}),
		postCssReporter({ clearReportedMessages: true })
	]);

	return gulp.src([ "pkg/web/assets/src/scss/**/*.scss" ]).
		pipe(cached('sasscache')).
		pipe(dependents()).
		pipe(gulpif(!argv.production, sourcemaps.init())).
		pipe(sass({ includePaths: sassPaths }).on("error", sass.logError)).
		pipe(localPostCSS).
		pipe(gulpif(argv.production, csso( { restructure:true } ) ) ).
		pipe(gulpif(!argv.production, sourcemaps.write())).
		pipe(gulp.dest("pkg/web/assets/dist/css/"));
};

const watchStyle = function()
{
	return gulp.watch( [ "pkg/web/assets/src/scss/**/*.scss" ], compileStyle );
};


const compileTasks = gulp.parallel(
	compileStyle,
	compileToES5,
	compileToES2015,
);

const watchTasks = gulp.parallel(
	gulp.series( compileStyle, watchStyle ),
	gulp.series( compileBothScripts, watchScript )
);

gulp.task( "default", compileTasks );
gulp.task( "compile", compileTasks );
gulp.task( "watch", watchTasks );
